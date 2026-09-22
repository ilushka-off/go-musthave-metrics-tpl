# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Бенчмарки

Бенчмарки для важнейших компонентов сервера (`internal/handler/metrics_bench_test.go`) измеряют обработку HTTP-запросов на обновление/чтение метрик через полный роутер (включая middleware):

```
go test ./internal/handler/... -bench=. -benchmem -run=^$
```

## Профилирование памяти

Профиль кучи снимался с работающего сервера (`net/http/pprof`, смонтирован на `/debug/pprof` через `chi/middleware.Profiler()`) во время нагрузки, сгенерированной `ab` (Apache Bench): одиночные обновления метрик через URL, батч-обновление (`/updates`, ~70 метрик за запрос) и чтение `/` (страница со всеми метриками).

Профили лежат в `profiles/base.pprof` (до оптимизации) и `profiles/result.pprof` (после).

### Что было найдено

Анализ `base.pprof` (`go tool pprof -top -sample_index=alloc_space profiles/base.pprof`) показал два неэффективных места в собственном коде:

1. `MetricsHandler.Index` — `fmt.Sprintf` вызывался в цикле по каждой метрике для построения HTML-страницы (58.6% всех аллокаций в профиле). Заменён на прямую запись частей строки в `strings.Builder` через `strconv.FormatFloat`/`strconv.FormatInt`.
2. `MetricsHandler.UpdateBatch` — срез имён метрик для события аудита рос через `append` без предварительной ёмкости, хотя итоговый размер уже известен (`len(metrics)`). Добавлена преаллокация `make([]string, 0, len(metrics))`.

### Результат: `pprof -top -diff_base=profiles/base.pprof profiles/result.pprof`

```
File: metrics-server
Type: alloc_space
Showing nodes accounting for -83.15MB, 10.30% of 807.36MB total
Dropped 8 nodes (cum <= 4.04MB)
Showing top 15 nodes out of 168
      flat  flat%   sum%        cum   cum%
  -77.50MB  9.60%  9.60%      -79MB  9.79%  fmt.Sprintf
  -45.50MB  5.64% 15.23%   -56.58MB  7.01%  internal/handler.(*MetricsHandler).Index
   32.68MB  4.05% 11.19%    32.68MB  4.05%  strings.(*Builder).WriteString (inline)
   17.74MB  2.20%  8.99%    17.74MB  2.20%  internal/repository.(*MemStorage).AllGauges
   13.50MB  1.67%  7.32%    13.50MB  1.67%  internal/strconv.FormatFloat
   -8.50MB  1.05%  8.37%   -15.56MB  1.93%  internal/handler.(*MetricsHandler).UpdateBatch
   -6.55MB  0.81%  9.18%    -6.55MB  0.81%  encoding/json/jsontext.(*decoderState).fetch
   -4.51MB  0.56%  9.74%    -4.51MB  0.56%  bufio.NewWriterSize (inline)
    4.50MB  0.56%  9.18%     4.50MB  0.56%  internal/repository.(*MemStorage).AllCounters
   -4.50MB  0.56%  9.74%    -4.50MB  0.56%  net/textproto.readMIMEHeader
      -4MB   0.5% 10.24%       -4MB   0.5%  context.(*cancelCtx).Done
   -3.50MB  0.43% 10.67%      -12MB  1.49%  net/http.(*conn).readRequest
   -3.01MB  0.37% 11.04%    -2.01MB  0.25%  encoding/json/v2.makeStringArshaler.func2
    3.01MB  0.37% 10.67%     3.01MB  0.37%  reflect.growslice
       3MB  0.37% 10.30%        3MB  0.37%  sync.(*Pool).pinSlow
```

`fmt.Sprintf` и `Index` — обе строки с отрицательным значением: `fmt.Sprintf` полностью ушёл из горячего пути (-77.5MB), а сам `Index` стал суммарно легче на 45.5MB (флэт) / 56.6MB (кумулятивно), несмотря на то что часть работы явно перешла в `strings.Builder.WriteString` и `strconv.FormatFloat` — итоговая сумма всё равно меньше, чем было в `fmt.Sprintf`. `UpdateBatch` также ушёл в минус (-8.5MB / -15.6MB) благодаря преаллокации среза. Остальные небольшие плюс/минус в рантайме и сетевом стеке (`context`, `net/http`, `sync.Pool`) — фоновый шум между двумя независимыми прогонами нагрузки, не связанный с внесёнными изменениями.

### Подтверждение через бенчмарки

```
до:    BenchmarkMetricsHandler_Index-8         83149   14334 ns/op   29686 B/op   337 allocs/op
после: BenchmarkMetricsHandler_Index-8        135484    8879 ns/op   23818 B/op   139 allocs/op

до:    BenchmarkMetricsHandler_UpdateBatch-8   32816   36230 ns/op   54314 B/op   254 allocs/op
после: BenchmarkMetricsHandler_UpdateBatch-8   32702   36080 ns/op   51622 B/op   247 allocs/op
```
