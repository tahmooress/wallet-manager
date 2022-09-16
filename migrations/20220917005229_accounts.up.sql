BEGIN;

CREATE TABLE IF NOT EXISTS accounts (
    user varchar(11) PRIMARY KEY NOT NULL,
    balance decimal(64,0)
    created_at TIMESTAMP(6) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP(6) NOT NULL
);

CREATE INDEX accounts_user_index ON accounts (user);

COMMIT;