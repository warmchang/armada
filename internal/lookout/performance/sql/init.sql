CREATE TABLE queue
(
    queue_id bigint       NOT NULL PRIMARY KEY,
    queue    varchar(512) NOT NULL
);

CREATE TABLE jobset
(
    jobset_id bigint        NOT NULL PRIMARY KEY,
    queue_id  bigint        NOT NULL,
    jobset    varchar(1024) NOT NULL
);

CREATE TABLE owner
(
    owner_id bigint       NOT NULL PRIMARY KEY,
    owner    varchar(512) NOT NULL
);

CREATE TABLE resources
(
    resources_id      bigint NOT NULL PRIMARY KEY,
    cpu               bigint NOT NULL,
    cpu_nano          bigint NOT NULL,
    memory            bigint NOT NULL,
    ephemeral_storage bigint NOT NULL,
    gpu               bigint NOT NULL
);

CREATE TABLE job
(
    job_id               varchar(32) NOT NULL PRIMARY KEY,
    queue_id             bigint      NOT NULL,
    owner_id             bigint      NOT NULL,
    jobset_id            bigint      NOT NULL,
    resources_id         bigint      NOT NULL,
    priority             float       NOT NULL,
    submitted            timestamp   NOT NULL,
    cancelled            timestamp   NULL,
    state                smallint    NOT NULL,
    last_transition_time timestamp   NOT NULL,
    job_spec             bytea       NOT NULL,
    duplicate            bool        NOT NULL DEFAULT false
);

CREATE TABLE cluster
(
    cluster_id bigint       NOT NULL PRIMARY KEY,
    cluster    varchar(512) NOT NULL
);

CREATE TABLE node
(
    node_id    bigint       NOT NULL PRIMARY KEY,
    cluster_id bigint       NOT NULL,
    node       varchar(512) NOT NULL
);

CREATE TABLE error
(
    error_id bigint NOT NULL PRIMARY KEY,
    error    bytea  NOT NULL
);

CREATE TABLE job_run
(
    run_id        varchar(36) NOT NULL PRIMARY KEY,
    job_id        varchar(32) NOT NULL,
    pod_number    int         NOT NULL,
    cluster_id    bigint      NOT NULL,
    node_id       bigint      NULL,
    leased        timestamp   NOT NULL,
    started       timestamp   NULL,
    finished      timestamp   NULL,
    job_run_state smallint    NOT NULL,
    error_id      bigint      NULL,
    exit_code     int         NULL,
    active        boolean     NOT NULL
);

CREATE TABLE job_run_container
(
    run_id         varchar(36)  NOT NULL,
    container_name varchar(512) NOT NULL,
    exit_code      int          NOT NULL,
    PRIMARY KEY (run_id, container_name)
);

CREATE TABLE annotation_keys
(
    key_id bigserial     NOT NULL PRIMARY KEY,
    key    varchar(1024) NOT NULL
);

CREATE TABLE user_annotation_lookup
(
    job_id varchar(32)   NOT NULL,
    key_id bigint        NOT NULL,
    value  varchar(1024) NOT NULL,
    PRIMARY KEY (job_id, key_id)
);
