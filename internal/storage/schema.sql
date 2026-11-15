CREATE TABLE IF NOT EXISTS links (
    code       TEXT        PRIMARY KEY,
    url        TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NULL,
    clicks     BIGINT      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
