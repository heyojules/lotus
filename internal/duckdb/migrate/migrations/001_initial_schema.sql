CREATE SEQUENCE IF NOT EXISTS logs_id_seq;

CREATE TABLE IF NOT EXISTS logs (
    id              BIGINT DEFAULT nextval('logs_id_seq'),
    timestamp       TIMESTAMP NOT NULL,
    orig_timestamp  TIMESTAMP,
    level           VARCHAR NOT NULL,
    level_num       INTEGER,
    message         VARCHAR NOT NULL,
    raw_line        VARCHAR,
    service         VARCHAR DEFAULT 'unknown',
    hostname        VARCHAR,
    pid             INTEGER,
    attributes      JSON,
    source          VARCHAR DEFAULT 'tcp'
);
