CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
                       id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                       email         citext NOT NULL UNIQUE,
                       password_hash text NOT NULL,
                       created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE income_sources (
                                id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                                user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                name            text NOT NULL,
                                amount          numeric(12,2) NOT NULL CHECK (amount > 0),
                                day_of_month    smallint NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
                                overflow_policy text NOT NULL DEFAULT 'forward' CHECK (overflow_policy IN ('forward','backward')),
                                archived_at     timestamptz,
                                created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_income_sources_user ON income_sources(user_id) WHERE archived_at IS NULL;

CREATE TABLE expense_obligations (
                                     id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                                     user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                     name            text NOT NULL,
                                     amount          numeric(12,2) NOT NULL CHECK (amount > 0),
                                     day_of_month    smallint NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
                                     overflow_policy text NOT NULL DEFAULT 'forward' CHECK (overflow_policy IN ('forward','backward')),
                                     archived_at     timestamptz,
                                     created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_expense_obligations_user ON expense_obligations(user_id) WHERE archived_at IS NULL;

CREATE TABLE savings_buckets (
                                 id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                                 user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                 name           text NOT NULL,
                                 target_amount  numeric(12,2) CHECK (target_amount IS NULL OR target_amount > 0),
                                 current_amount numeric(12,2) NOT NULL DEFAULT 0,
                                 created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_savings_buckets_user ON savings_buckets(user_id);

CREATE TABLE savings_rules (
                               id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                               income_source_id uuid NOT NULL REFERENCES income_sources(id) ON DELETE CASCADE,
                               bucket_id        uuid NOT NULL REFERENCES savings_buckets(id) ON DELETE CASCADE,
                               mode             text NOT NULL CHECK (mode IN ('fixed','percent')),
                               value            numeric(12,4) NOT NULL CHECK (
                                   (mode = 'percent' AND value > 0 AND value <= 1)
                                       OR (mode = 'fixed'   AND value > 0)
                                   ),
                               created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_savings_rules_income_source ON savings_rules(income_source_id);
CREATE INDEX idx_savings_rules_bucket ON savings_rules(bucket_id);

CREATE TABLE balance_snapshots (
                                   id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                                   user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                   amount     numeric(12,2) NOT NULL,
                                   as_of_date date NOT NULL,
                                   created_at timestamptz NOT NULL DEFAULT now(),
                                   UNIQUE (user_id, as_of_date)
);
CREATE INDEX idx_balance_snapshots_latest ON balance_snapshots(user_id, as_of_date DESC);

CREATE TABLE transactions (
                              id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
                              user_id               uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                              type                  text NOT NULL CHECK (type IN ('income','expense','saving','adhoc')),
                              amount                numeric(12,2) NOT NULL CHECK (amount > 0),
                              txn_date              date NOT NULL,
                              status                text NOT NULL CHECK (status IN ('planned','confirmed','manual')),
                              income_source_id      uuid REFERENCES income_sources(id) ON DELETE SET NULL,
                              expense_obligation_id uuid REFERENCES expense_obligations(id) ON DELETE SET NULL,
                              savings_rule_id       uuid REFERENCES savings_rules(id) ON DELETE SET NULL,
                              note                  text,
                              created_at            timestamptz NOT NULL DEFAULT now(),
                              confirmed_at          timestamptz
);
CREATE INDEX idx_transactions_user_date ON transactions(user_id, txn_date);
CREATE INDEX idx_transactions_user_status ON transactions(user_id, status);
