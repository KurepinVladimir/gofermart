package orders

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/KurepinVladimir/gofermart/internal/accrual"
	"github.com/KurepinVladimir/gofermart/internal/logger"
)

type Worker struct {
	repo interface {
		PickPendingOrders(ctx context.Context, limit int) ([]string, error)
		MarkProcessing(ctx context.Context, number string) error
		MarkInvalid(ctx context.Context, number string) error
		ApplyAccrualByNumber(ctx context.Context, number string, accrual float64) error
	}
	client *accrual.Client
	limit  int           // сколько номеров за тик опрашивать
	tick   time.Duration // период опроса
}

func NewWorker(repo interface {
	PickPendingOrders(ctx context.Context, limit int) ([]string, error)
	MarkProcessing(ctx context.Context, number string) error
	MarkInvalid(ctx context.Context, number string) error
	ApplyAccrualByNumber(ctx context.Context, number string, accrual float64) error
}, client *accrual.Client, limit int, tick time.Duration) *Worker {
	if limit <= 0 {
		limit = 20
	}
	if tick <= 0 {
		tick = 2 * time.Second
	}
	return &Worker{repo: repo, client: client, limit: limit, tick: tick}
}

func (w *Worker) Run(ctx context.Context) {
	if w.client == nil {
		logger.Log.Warn("accrual worker disabled: no client")
		return
	}
	ticker := time.NewTicker(w.tick)
	defer ticker.Stop()

	var pauseUntil time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// если нас попросили подождать Retry-After
			if time.Now().Before(pauseUntil) {
				continue
			}

			numbers, err := w.repo.PickPendingOrders(ctx, w.limit)
			if err != nil {
				logger.Log.Warn("pick pending orders failed", zap.Error(err))
				continue
			}
			for _, num := range numbers {
				// запрос в внешнюю систему
				resp, code, retryAfter, err := w.client.GetInfo(num)
				if err != nil {
					logger.Log.Warn("accrual get failed", zap.String("number", num), zap.Error(err))
					continue
				}
				if code == 429 {
					// глобальная пауза
					if retryAfter > 0 {
						pauseUntil = time.Now().Add(retryAfter)
						logger.Log.Info("429 received", zap.Duration("retry_after", retryAfter))
					}
					break // выйдем из цикла обработки, подождём следующий тик
				}
				if code == 204 {
					// в расчёт ещё не был взят — оставим как есть
					continue
				}
				if code != 200 {
					// временная ошибка — попробуем потом
					continue
				}
				// 200 OK: смотрим статус
				switch resp.Status {
				case "REGISTERED", "PROCESSING":
					_ = w.repo.MarkProcessing(ctx, num)
				case "INVALID":
					_ = w.repo.MarkInvalid(ctx, num)
				case "PROCESSED":
					var accrualValue float64
					if resp.Accrual != nil {
						accrualValue = *resp.Accrual
					}
					if err := w.repo.ApplyAccrualByNumber(ctx, num, accrualValue); err != nil {
						logger.Log.Warn("apply accrual failed", zap.String("number", num), zap.Error(err))
					}
				default:
					// неизвестный статус — пропустим
				}
			}
		}
	}
}
