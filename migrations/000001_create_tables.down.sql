-- Миграция 000001 (откат): Удаление всех таблиц
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS partners;
DROP TABLE IF EXISTS job_materials;
DROP TABLE IF EXISTS materials;
DROP TABLE IF EXISTS job_events;
DROP TABLE IF EXISTS jobs;
DROP EXTENSION IF EXISTS "pgcrypto";