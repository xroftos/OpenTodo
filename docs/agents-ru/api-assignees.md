---
title: Исполнители задач: REST и MCP (по-русски)
summary: Готовые примеры curl и MCP-вызовов для работы с исполнителями задач: список участников, создание и изменение задач с assignee_ids, assignees, ошибка not_a_member.
audience: developer
section: Guides
order: 81
---

# Исполнители задач: REST и MCP

Задача несёт список исполнителей **`assignee_ids`** — массив id участников. Это ломающее изменение: прежнее единичное поле `assignee_id` больше не принимается. В MCP-инструментах `create_tasks`/`update_tasks` то же самое поле называется **`assignees`** и принимает id участника **или точное отображаемое имя**. Важно: агент-токен (scope `agent`) назначает исполнителей **только через MCP-инструменты**; на обычных REST-маршрутах `POST`/`PATCH /api/v1/tasks` поле `assignee_ids` с агент-токеном отклоняется (`403 insufficient_scope`) — с обычным пользовательским токеном REST-примеры ниже работают как есть.

> Примечание: [docs/api.md](../api.md) в момент написания мог ещё описывать старое поле `assignee_id`; на этой ветке его заменил список `assignee_ids`. Этот документ — свод правил и примеров для агент-токенов; аутентификация и маршруты описаны в [docs/api.md](../api.md#authentication) и [docs/mcp.md](../mcp.md).

Во всех примерах ниже: `$BASE` — адрес сервера (для набора `deploy/family` — `http://localhost:4002`), `$T` — токен (`Authorization: Bearer ot_…`). Агентский токен (scope `agent`) читает всё, что видит его владелец, и пишет только на агентских маршрутах и только когда включены оба переключателя записи (владельца и участника); обычный `full`-токен тоже подходит для примеров, но агенту выдавайте именно `agent` (см. [AGENTS.md](AGENTS.md)).

## 1. Список участников проекта

Исполнителями могут быть только участники проекта задачи, поэтому сначала получите их id:

```bash
curl -sS -H "Authorization: Bearer $T" "$BASE/api/v1/projects/<project_id>/members"
```

```json
{
  "projectId": "<project_id>",
  "role": "owner",
  "ownerId": "<alice>",
  "members": [
    { "userId": "<alice>", "name": "Мама",  "email": "mama@example.com", "role": "owner" },
    { "userId": "<bob>",   "name": "Папа",  "email": "papa@example.com", "role": "admin" },
    { "userId": "<carol>", "name": "Аня",   "email": "anya@example.com", "role": "member" },
    { "userId": "<dave>",  "name": "Миша",  "email": "misha@example.com", "role": "member" }
  ]
}
```

Тот же список агент видит через MCP-инструмент `list_projects` (поля участников — в ответе для общих проектов) или `list_workspaces` для рабочих пространств.

## 2. Создать задачу с исполнителями

```bash
curl -sS -X POST "$BASE/api/v1/tasks" \
  -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
  -d '{
    "title": "Полить помидоры",
    "project_id": "<project_id>",
    "due": { "date": "2026-11-15" },
    "assignee_ids": ["<carol>", "<dave>"]
  }'
```

Ответ — обычная задача с `"assignee_ids":["<carol>","<dave>"]`; каждому назначенному приходит уведомление в приложении. Поле заменяет список целиком: создатели задачи не обязаны быть исполнителями.

## 3. Сменить исполнителей

`PATCH /api/v1/tasks/{id}` записывает список целиком — то, что вы прислали, и есть новый полный состав:

```bash
curl -sS -X PATCH "$BASE/api/v1/tasks/<task_id>" \
  -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
  -d '{ "assignee_ids": ["<bob>"] }'
```

## 4. Назначить нескольких

Передайте все id сразу — порядок не важен, дубликаты не нужны:

```bash
curl -sS -X PATCH "$BASE/api/v1/tasks/<task_id>" \
  -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
  -d '{ "assignee_ids": ["<alice>", "<bob>", "<carol>"] }'
```

## 5. Снять всех исполнителей

Пустой массив — «исполнителей нет»:

```bash
curl -sS -X PATCH "$BASE/api/v1/tasks/<task_id>" \
  -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
  -d '{ "assignee_ids": [] }'
```

## 6. Ошибка not_a_member

Назначение человека, который не состоит в проекте задачи (или вас самого — на личной задаче без проекта), отказывается целиком и ничего не записывает:

```bash
curl -sS -X PATCH "$BASE/api/v1/tasks/<task_id>" \
  -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
  -d '{ "assignee_ids": ["<stranger>"] }'
# 400 {"error":{"code":"not_a_member","message":"task.assignee_ids: the assignee must be a member of the task's project (or yourself for a personal task)"}}
```

На синхронизационном эндпоинте та же операция отклоняется по одному элементу: `rejected: [{"id":"…","reason":"not_a_member"}]`. Правило то же, что и у прежнего `assignee_id`; зритель (viewer) общего проекта назначать не может — `403 forbidden`, без учётных данных — `401 unauthorized`.

## 7. MCP: create_tasks и update_tasks с assignees

На MCP те же поля зовутся `assignees` и, кроме id, принимают **точное отображаемое имя** участника. Инструменты `create_tasks` и `update_tasks` доступны агент-токену, когда включены оба переключателя записи:

```json
{"name": "create_tasks", "arguments": {"items": [
  {"title": "Собрать портфель", "project_id": "<project_id>",
   "due": {"date": "2026-11-15"}, "assignees": ["Аня"]}
], "client_request_id": "semya-1"}}

{"name": "update_tasks", "arguments": {"updates": [
  {"id": "<task_id>", "assignees": ["Папа", "<carol>"]}
]}}
```

Имя сравнивается с отображаемым именем участника проекта дословно; если под именем никого нет или их несколько, используйте id из `list_projects` / списка участников — вызов вернёт `not_a_member` или ошибку `-32602`, и ничего не изменится. При смене исполнителей через `update_tasks` соблюдайте правило агента: не снимать исполнителя без подтверждения человека и не удалять задачи — помечать (см. [AGENTS.md](AGENTS.md#5-исполнители-assignee_ids-и-assignees)).

## 8. Памятка

- `Authorization: Bearer ot_…` — заголовок для каждого запроса; агентский токен scope `agent`.
- REST-поле — `assignee_ids` (массив id; только пользовательский токен); MCP-поле — `assignees` (id или точные имена; единственный путь для агент-токена).
- Чтение участников — `GET /api/v1/projects/{id}/members` (маршрут уровня «member»).
- Не участник проекта → `400 not_a_member`, запись не выполняется.
- Назначение создаёт уведомление каждому новому исполнителю; список «Мои задачи» (вид `assigned`) собирает всё назначенное участнику.
- Лимиты агент-токена: 30 вызовов записи в минуту, 1 000 изменённых задач в сутки; при превышении `429 rate_limited` с `Retry-After`.
