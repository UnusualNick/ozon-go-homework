-- ДЗ 4. Все пять обязательных задач решены запросами для PostgreSQL.

-- 1. Самый старший пользователь Москвы. При одинаковом возрасте возвращаем
-- всех кандидатов; на тестовых данных это Борис и Вера.
SELECT name, age
FROM users
WHERE city = 'Москва'
  AND age = (SELECT MAX(age) FROM users WHERE city = 'Москва')
ORDER BY name;

-- 2. Города, в которых больше одного пользователя.
SELECT city, COUNT(*) AS user_count
FROM users
GROUP BY city
HAVING COUNT(*) > 1
ORDER BY user_count DESC, city ASC;

-- 3. Сумма заказов каждого пользователя, включая тех, кто ничего не купил.
-- Группируем по id, чтобы однофамильцы не сливались в одну строку.
SELECT u.name, COALESCE(SUM(o.price), 0) AS total_spent
FROM users AS u
LEFT JOIN orders AS o ON o.user_id = u.id
GROUP BY u.id, u.name
ORDER BY total_spent DESC, u.name ASC, u.id ASC;

-- 4. Имена покупателей наушников без повторов.
SELECT DISTINCT u.name
FROM users AS u
JOIN orders AS o ON o.user_id = u.id
WHERE o.item = 'наушники'
ORDER BY u.name;

-- 5. Все заказы с минимальной ценой среди заказов дороже 10000.
-- DISTINCT здесь нет: два одинаковых заказа тоже должны дать две строки.
SELECT item, price
FROM orders
WHERE price > 10000
  AND price = (SELECT MIN(price) FROM orders WHERE price > 10000)
ORDER BY item, id;
