BEGIN;
DELETE FROM users WHERE email LIKE '%@musicroom.local' OR email LIKE '%@musicroom.bulk';
COMMIT;
