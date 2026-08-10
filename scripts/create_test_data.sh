#!/bin/bash

# Базовый URL API
BASE_URL="http://localhost:8080/api"

# Сначала регистрируем пользователя и получаем токен
echo "=== Регистрация пользователя ==="
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }')

echo "Register response: $REGISTER_RESPONSE"

# Пробуем залогиниться (если пользователь уже существует)
echo ""
echo "=== Логин ==="
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }')

echo "Login response: $LOGIN_RESPONSE"

# Извлекаем токен (предполагается, что ответ содержит access_token)
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Не удалось получить токен. Проверьте регистрацию/логин."
  exit 1
fi

echo ""
echo "=== Токен получен ==="
echo "Token: ${TOKEN:0:50}..."

# Функция для выполнения запросов с авторизацией
auth_curl() {
  curl -s -X "$1" "$2" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "$3"
}

echo ""
echo "=== Создание источников дохода ==="

# Доход 1: каждое 5 число - 100к рублей
INCOME1=$(auth_curl POST "$BASE_URL/income-sources" '{
  "name": "Основная зарплата",
  "amount": 100000,
  "day_of_month": 5,
  "overflow_policy": "forward"
}')
echo "Income 1 (5th, 100k): $INCOME1"
INCOME1_ID=$(echo $INCOME1 | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Income 1 ID: $INCOME1_ID"

# Доход 2: каждое 7 число - 30к
INCOME2=$(auth_curl POST "$BASE_URL/income-sources" '{
  "name": "Подработка",
  "amount": 30000,
  "day_of_month": 7,
  "overflow_policy": "forward"
}')
echo "Income 2 (7th, 30k): $INCOME2"
INCOME2_ID=$(echo $INCOME2 | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Income 2 ID: $INCOME2_ID"

# Доход 3: каждое 20 число - 100к
INCOME3=$(auth_curl POST "$BASE_URL/income-sources" '{
  "name": "Проектная работа",
  "amount": 100000,
  "day_of_month": 20,
  "overflow_policy": "forward"
}')
echo "Income 3 (20th, 100k): $INCOME3"
INCOME3_ID=$(echo $INCOME3 | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Income 3 ID: $INCOME3_ID"

echo ""
echo "=== Создание обязательных расходов ==="

# Расход 1: 5 число - 30к
EXPENSE1=$(auth_curl POST "$BASE_URL/expense-obligations" '{
  "name": "Аренда",
  "amount": 30000,
  "day_of_month": 5,
  "overflow_policy": "forward"
}')
echo "Expense 1 (5th, 30k): $EXPENSE1"

# Расход 2: 28 число - 32к
EXPENSE2=$(auth_curl POST "$BASE_URL/expense-obligations" '{
  "name": "Коммуналка и связь",
  "amount": 32000,
  "day_of_month": 28,
  "overflow_policy": "forward"
}')
echo "Expense 2 (28th, 32k): $EXPENSE2"

echo ""
echo "=== Создание накопительных целей (savings buckets) ==="

# Создаем бакеты для накоплений
BUCKET1=$(auth_curl POST "$BASE_URL/savings-buckets" '{
  "name": "Подушка безопасности",
  "target_amount": 500000
}')
echo "Bucket 1: $BUCKET1"
BUCKET1_ID=$(echo $BUCKET1 | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Bucket 1 ID: $BUCKET1_ID"

BUCKET2=$(auth_curl POST "$BASE_URL/savings-buckets" '{
  "name": "Отпуск",
  "target_amount": 150000
}')
echo "Bucket 2: $BUCKET2"
BUCKET2_ID=$(echo $BUCKET2 | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Bucket 2 ID: $BUCKET2_ID"

echo ""
echo "=== Создание правил накопления (savings rules) ==="

# Правило: 10% от основной зарплаты на подушку безопасности
if [ -n "$INCOME1_ID" ] && [ -n "$BUCKET1_ID" ]; then
  RULE1=$(auth_curl POST "$BASE_URL/savings-rules" "{
    \"income_source_id\": \"$INCOME1_ID\",
    \"bucket_id\": \"$BUCKET1_ID\",
    \"mode\": \"percent\",
    \"value\": 0.1
  }")
  echo "Rule 1 (10% from main income to emergency fund): $RULE1"
fi

# Правило: 5000 фиксировано от подработки на отпуск
if [ -n "$INCOME2_ID" ] && [ -n "$BUCKET2_ID" ]; then
  RULE2=$(auth_curl POST "$BASE_URL/savings-rules" "{
    \"income_source_id\": \"$INCOME2_ID\",
    \"bucket_id\": \"$BUCKET2_ID\",
    \"mode\": \"fixed\",
    \"value\": 5000
  }")
  echo "Rule 2 (5k fixed from side income to vacation): $RULE2"
fi

echo ""
echo "=== Готово! ==="
echo "Созданы:"
echo "- 3 источника дохода (5-е: 100к, 7-е: 30к, 20-е: 100к)"
echo "- 2 обязательных расхода (5-е: 30к, 28-е: 32к)"
echo "- 2 накопительные цели"
echo "- 2 правила накопления"
