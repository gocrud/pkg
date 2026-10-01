DROP TABLE IF EXISTS undo_log;
CREATE TABLE undo_log
(
    id            bigserial    NOT NULL,
    branch_id     bigint       NOT NULL,
    xid           text         NOT NULL,
    context       text         NOT NULL,
    rollback_info bytea        NOT NULL,
    log_status    int          NOT NULL,
    log_created   timestamp    NOT NULL,
    log_modified  timestamp    NOT NULL,
    ext           text         DEFAULT NULL,
    PRIMARY KEY (id),
    CONSTRAINT ux_undo_log UNIQUE (xid, branch_id)
);

CREATE INDEX ix_log_created ON undo_log (log_created);
