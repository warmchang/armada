CREATE TABLE job
(
    job_id               varchar(32)      NOT NULL PRIMARY KEY,
    queue                varchar(512)     NOT NULL,
    owner                varchar(512)     NOT NULL,
    jobset               varchar(1024)    NOT NULL,
    cpu                  bigint           NOT NULL,
    memory               bigint           NOT NULL,
    ephemeral_storage    bigint           NOT NULL,
    gpu                  bigint           NOT NULL,
    priority             double precision NOT NULL,
    submitted            timestamp        NOT NULL,
    cancelled            timestamp        NULL,
    state                smallint         NOT NULL,
    last_transition_time timestamp        NOT NULL,
    job_spec             bytea            NOT NULL,
    duplicate            bool             NOT NULL DEFAULT false
);

CREATE TABLE job_run
(
    run_id        varchar(36)  NOT NULL PRIMARY KEY,
    job_id        varchar(32)  NOT NULL,
    pod_number    int          NOT NULL,
    cluster       varchar(512) NOT NULL,
    node          varchar(512) NULL,
    leased        timestamp    NOT NULL,
    started       timestamp    NULL,
    finished      timestamp    NULL,
    job_run_state smallint     NOT NULL,
    error         bytea        NULL,
    exit_code     int          NULL,
    active        boolean      NOT NULL
);

CREATE TABLE user_annotation_lookup
(
    job_id varchar(32)   NOT NULL,
    key    varchar(1024) NOT NULL,
    value  varchar(1024) NOT NULL,
    PRIMARY KEY (job_id, key)
);

CREATE INDEX idx_job_queue ON job (queue);
CREATE INDEX idx_job_queue_pattern ON job (queue varchar_pattern_ops);

CREATE INDEX idx_job_jobset ON job (jobset);
CREATE INDEX idx_job_jobset_pattern ON job (jobset varchar_pattern_ops);

CREATE INDEX idx_job_run_node ON job_run (node);
CREATE INDEX idx_job_run_node_pattern ON job_run (node varchar_pattern_ops);
