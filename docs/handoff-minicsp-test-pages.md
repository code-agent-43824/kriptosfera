# Handoff: деплой тест-страниц для диагностики Mini CSP (early-flag)

Адресат: агент, который ведёт основной код и управляет серверами (в т.ч.
`mescheryakov.pro` / VDSina). Автор: диагностический агент, работавший в
веб-сессии Claude Code **без доступа к SSH** (порт 22 закрыт сетевой политикой
песочницы; наружу доступны только 80/443). Поэтому правки в репозиторий я залил
сам в `main`, а **деплой тест-страниц на сервер прошу выполнить тебя** — ниже
точная задача с готовым кодом.

Дата: 2026-05-31.

Update from Watson, 2026-05-31: `internal-csp-early` has been deployed to the
project static server, and the hosted diagnostics page has been refreshed from
`payload-template/diagnostics/diagnostics.html`.

---

## Часть 1. Что изменилось в репозитории без твоего участия

Все изменения уже в `main` (CI `build-windows` зелёный). Кратко, по коммитам:

1. **Сборка из коробки + hardening** (`6afed62`)
   - Закоммичены нулевые placeholder `internal/bootstrap/payload.zip` и
     `cryptopro-plugin.zip`, чтобы `go build`/`go test ./...` работали на чистом
     checkout (build-скрипты перезаписывают их реальными артефактами). Пустой
     embed трактуется как «bundle не встроен».
   - `validateAppConfig` теперь проверяет `profileName` как безопасный одиночный
     сегмент пути (анти-traversal).
   - Лимит размера remote-загрузки (pinned size / 1 GiB) с ранним обрывом.

2. **Документация + лицензия** (`25da90c`)
   - Apache-2.0 `LICENSE` + `NOTICE` (сторонние runtime-компоненты сохраняют
     свои условия и в Git не хранятся).
   - `CONTRIBUTING.md`, `docs/README.md` (индекс), `docs/architecture.md`
     (поток launcher + раскладка AppData), package-level godoc, `CHANGELOG.md`.
   - Весь Go-код приведён к `gofmt` (только форматирование).

3. **Раздельные Windows-артефакты** (`5be55ba`)
   - `build-windows` публикует **два** независимых артефакта:
     `kriptosfera-windows-embedded` и `kriptosfera-windows-remote`, чтобы тонкий
     launcher качался без большого embedded-файла.

4. **CI-фикс** (`54dff4d`)
   - `TestCryptoProPluginManagerSkipsInvalidMSIPseudoPaths` сделан
     Windows-переносимым (не использует `os.IsNotExist` для пути с `:`).
   - `build-launcher.ps1` теперь **валит шаг при падении `go test`** (раньше
     тест-фейл молча проскакивал).

5. **Перф/UX/ревью** (`6a025e9`)
   - Reuse payload без полного пере-хэширования на каждом старте (только
     проверка наличия файлов; полный SHA-256 — при распаковке).
   - `validateCryptoProPluginLayout` — один обход дерева вместо 11.
   - Native messaging не переписывает манифест и не зовёт `reg.exe` повторно,
     если ничего не изменилось (state-файл).
   - Второй запуск приложения больше не падает с «bootstrap already in
     progress»: reuse проверяется до лока, лок ждёт (bounded) и heartbeat'ится.
   - CI-тест проверяет, что вшитый CryptoPro bundle содержит все required-файлы.

6. **Диагностика Mini CSP** (`9ea6006`) — см. Часть 2 (это контекст задачи).

> Если что-то из этого конфликтует с твоими планами — скажи, обсудим. Внешнее
> поведение launcher не менялось, кроме осознанного UX двойного запуска.

---

## Часть 2. Ключевой вывод расследования по Mini CSP (по фактам)

Цель: понять, **почему на чистой машине без системного CSP внутренний Mini CSP
не активируется** (провайдеры не перечисляются, `CSPName(80)` → `0x80090017`).

Проведён статический анализ **реальных** бинарей из нашего
`cryptopro-plugin.zip` (2.0.15700) и развёрнут эталонный CryptoPro CSP Lite на
Linux. Доказательно **сняты** прежние гипотезы:

- **Файлы/раскладка полные** — совпадают с реальной установкой `addminicsp`
  один-в-один (корень + `Mini CSP`).
- **Реестр не нужен** — `cpsuprt.dll` читает `config.ini` через собственную
  абстракцию `support_registry_*`; на машине с `addminicsp` ветки `Crypto Pro`
  в реестре нет вообще (подтверждено вручную).
- **Лицензия в комплекте** — `ProductID` в `Mini CSP\license.ini`, `npcades.dll`
  читает его оттуда (`\local\license\ProductID\{50F91F80-...}`).
- **Доп. рантайм не нужен** — Mini CSP DLL импортят только
  `KERNEL32/ADVAPI32/msvcrt/ntdll`.
- **Разрядность** — `nmcades.exe`/`npcades.dll` 32-битные → грузят 32-битный
  `Mini CSP\capi20.dll` → читают `config.ini` (не `config64.ini`), где провайдеры
  75/80/81 уже описаны (`Image Path = cpcspi.dll`).

**Подтверждённый механизм активации** (строки в `npcades.dll`): рядом лежат
`result = cadesplugin.EnableInternalCSP`, `Mini CSP\capi20.dll`,
`GetModuleFileNameW`, `LoadLibraryExA(capi20.dll) failed.`, `AddAvailableCsps`.
То есть: **npcades спрашивает у страницы значение `cadesplugin.EnableInternalCSP`
рано (callback'ом), и если оно `true` — грузит `Mini CSP\capi20.dll`
module-relative путём и перечисляет провайдеры из `config.ini`.** Реестр,
wrapper-хост и «уплощение» папки НЕ требуются.

**Главное подозрение (гипотеза A — тайминг флага):** существующие тест-страницы
(`internal-csp`, `internal-csp-aggressive`) ставят флаг **после/около** загрузки
`cadesplugin_api.js`. Если npcades читает флаг раньше, он видит
`false`/`undefined` и **вообще не грузит Mini CSP**. Нужно поставить флаг
**максимально рано** (до `cadesplugin_api.js`) и удерживать его, затем сравнить.

Полные детали — в `docs/cryptopro-csp-lite-plan.md` (раздел «Ground truth from
binary analysis»).

---

## Часть 3. ЗАДАЧА для тебя — выложить новую тест-страницу `internal-csp-early`

На сервере уже задеплоены три варианта:

```
https://mescheryakov.pro/kriptosfera/cryptopro-cades-test/vanilla/demopage/cades_bes_sample.html
https://mescheryakov.pro/kriptosfera/cryptopro-cades-test/internal-csp/demopage/cades_bes_sample.html
https://mescheryakov.pro/kriptosfera/cryptopro-cades-test/internal-csp-aggressive/demopage/cades_bes_sample.html
https://mescheryakov.pro/kriptosfera/cryptopro-cades-test/internal-csp-early/demopage/cades_bes_sample.html
```

Четвёртый вариант `internal-csp-early` идентичен `internal-csp`, но выставляет
флаг **до** `../cadesplugin_api.js` и переутверждает его ~5 сек.

### Шаги

1. Done: скопирован каталог `internal-csp` в `internal-csp-early` (вся `demopage/` со
   всеми относительными скриптами `Code.js`, `lights.js`, `load_extension.js`,
   `../cadesplugin_api.js`, `../es6-promise.min.js` и т.д. — структура та же).

2. Done: в `internal-csp-early/demopage/cades_bes_sample.html` внесены два изменения.

   **(a) Убери** старый поздний блок (он стоит сразу ПОСЛЕ строки с
   `../cadesplugin_api.js`):

   ```html
   <script language="javascript">
       cadesplugin.EnableInternalCSP = true;
       window.postMessage("EnableInternalCSP=true", "*");
   </script>
   ```

   **(b) Вставь РАНО — непосредственно ПЕРЕД** строкой
   `<script ... src="../cadesplugin_api.js?v=313061"></script>` — такой блок:

   ```html
   <!-- internal-csp-early: set EnableInternalCSP BEFORE cadesplugin_api.js so
        nmcades/npcades sees it when it first reads the flag via callback -->
   <script language="javascript">
       window.cadesplugin = window.cadesplugin || {};
       window.cadesplugin.EnableInternalCSP = true;
       window.postMessage("EnableInternalCSP=true", "*");
       window.__csp_early_flag_set_at__ = Date.now();
   </script>
   ```

   **(c) И сразу ПОСЛЕ** строки с `../cadesplugin_api.js` добавь переутверждение
   (на случай поздней загрузки `content.js`/`nmcades_plugin_api.js`):

   ```html
   <script language="javascript">
       try { window.cadesplugin.EnableInternalCSP = true; } catch (e) {}
       window.postMessage("EnableInternalCSP=true", "*");
       (function () {
           var n = 0;
           var id = setInterval(function () {
               try { window.cadesplugin.EnableInternalCSP = true; } catch (e) {}
               window.postMessage("EnableInternalCSP=true", "*");
               if (++n >= 20) clearInterval(id); // ~5s
           }, 250);
       }());
   </script>
   ```

   Порядок в итоге: `... lights.js` → **блок (b)** → `../cadesplugin_api.js` →
   **блок (c)** → `if (ShowMoreListener) ShowMoreListener();` → `load_extension.js`.

3. Done: каталог выложен по тому же базовому пути:
   ```
   https://mescheryakov.pro/kriptosfera/cryptopro-cades-test/internal-csp-early/demopage/cades_bes_sample.html
   ```
   Права/владелец — как у соседних `cryptopro-cades-test/*` (тот же web-root).
   Иначе ничего настраивать не нужно: все скрипты относительные.

Verified: the page returns HTTP 200 and the public HTML shows the early
pre-`cadesplugin_api.js` flag block followed by the post-load reassertion loop.

### Как проверить (на чистой Windows-машине без системного CSP)

Открой в нашем Chromium-окружении (с загруженным extension + native host) по
очереди `internal-csp` и `internal-csp-early`, в каждом — стандартный поток
демо-страницы (выбор сертификата / подпись). Сравни:

- **Если на `internal-csp-early` провайдеры начали перечисляться, а на
  `internal-csp` — нет** → подтверждена гипотеза A (тайминг флага). Тогда мы
  переносим ранний-флаг паттерн в продакшн (extension/диагностику) — этот вывод
  закроет этап 6 по части активации.
- **Если оба молчат одинаково** → флаг не при чём (гипотеза B/C): нужен ProcMon
  по `nmcades.exe` — фильтр `Load Image` на `Mini CSP\capi20.dll` и `CreateFile`
  на `config.ini`/`asn1*.dll`, искать `NAME NOT FOUND`/`PATH NOT FOUND`.

> Доп. сигнал без ProcMon: наша обновлённая `payload-template/diagnostics/
> diagnostics.html` (коммит `9ea6006`) уже ставит флаг рано и печатает явный
> вердикт A/B/C. Если удобнее — задеплой ИЛИ её рядом и сверь вердикт.

### Опционально, но полезно

Done: обновлён deployed `diagnostics.html` из репозитория
(`payload-template/diagnostics/diagnostics.html`) — там уже есть ранний флаг,
таймлайн значения и автоматический вердикт A/B/C. Это удобнее для отладки, чем
ручное сравнение страниц.

Verified URL:

```text
https://mescheryakov.pro/kriptosfera/diagnostics/diagnostics.html
```

---

## Часть 4. Что НЕ нужно делать (выводы расследования)

Чтобы не тратить время на уже отвергнутое фактами:

- ❌ Не писать ничего в реестр (Mini CSP его не использует).
- ❌ Не «уплощать» Mini CSP в корень плагина (механизм рассчитан на подпапку
  `Mini CSP`, путь module-relative).
- ❌ Не делать wrapper-хост.
- ❌ Не искать недостающие лицензию/рантайм/файлы — их хватает.

Фокус — **только** на доставке флага `EnableInternalCSP` вовремя и, если это не
поможет, на загрузке `Mini CSP\capi20.dll` и его зависимостей (ProcMon).

---

Вопросы/результаты прогона можно зафиксировать дополнением к этому файлу или в
`docs/cryptopro-csp-lite-plan.md`.
