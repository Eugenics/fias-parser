create table migrations
(
    id         bigserial
        primary key,
    name       text                                   not null
        unique,
    created_at timestamp with time zone default now() not null
);

create index name_idx
    on migrations (name);

create table params
(
    id            integer not null
        constraint pk_params
            primary key,
    object_id     integer not null,
    change_id     integer,
    change_id_end integer,
    type_id       smallint,
    value         text,
    update_date   date,
    start_date    date,
    end_date      date
);

create index idx_params_object_id
    on params (object_id);

create index idx_params_type_id
    on params (type_id);

create table address_objects
(
    id           integer               not null
        constraint pk_address_objects
            primary key,
    object_id    integer               not null,
    object_guid  uuid                  not null,
    change_id    integer,
    name         varchar(255),
    type_name    varchar(100),
    level        smallint,
    oper_type_id smallint,
    prev_id      integer,
    next_id      integer,
    update_date  date,
    start_date   date,
    end_date     date,
    is_actual    boolean default false not null,
    is_active    boolean default false not null
);

create index idx_address_objects_object_guid
    on address_objects (object_guid);

create index idx_address_objects_level
    on address_objects (level);

create index idx_address_objects_is_actual
    on address_objects (is_actual);

create index idx_address_objects_is_active
    on address_objects (is_active);

create table object_levels
(
    level       integer not null
        primary key,
    name        varchar(255),
    start_date  date,
    end_date    date,
    update_date date,
    is_active   boolean
);

create table house_types
(
    id          integer not null
        primary key,
    name        varchar(255),
    short_name  varchar(50),
    "desc"      varchar(255),
    is_active   boolean,
    update_date date,
    start_date  date,
    end_date    date
);

create table address_object_types
(
    id          integer not null
        primary key,
    level       integer,
    name        varchar(255),
    short_name  varchar(50),
    "desc"      varchar(255),
    update_date date,
    start_date  date,
    end_date    date,
    is_active   boolean
);

create table apartment_types
(
    id          integer not null
        primary key,
    name        varchar(255),
    short_name  varchar(50),
    "desc"      varchar(255),
    is_active   boolean,
    start_date  date,
    end_date    date,
    update_date date
);

create table ndoc_kinds
(
    id   integer not null
        primary key,
    name varchar(255)
);

create table ndoc_types
(
    id         integer not null
        primary key,
    name       varchar(255),
    start_date date,
    end_date   date
);

create table operation_types
(
    id          integer not null
        primary key,
    name        varchar(255),
    is_active   boolean,
    update_date date,
    start_date  date,
    end_date    date
);

create table param_types
(
    id          integer not null
        primary key,
    name        varchar(255),
    "desc"      varchar(255),
    code        varchar(50),
    is_active   boolean,
    update_date date,
    start_date  date,
    end_date    date
);

create table room_types
(
    id          integer not null
        primary key,
    name        varchar(255),
    "desc"      varchar(255),
    is_active   boolean,
    start_date  date,
    end_date    date,
    update_date date
);

create table adm_hierarchy
(
    id            integer not null
        primary key,
    object_id     integer,
    parent_obj_id integer,
    change_id     integer,
    region_code   integer,
    prev_id       integer,
    next_id       integer,
    update_date   date,
    start_date    date,
    end_date      date,
    is_active     boolean,
    path          varchar(500)
);

create table version_info
(
    id                serial
        primary key,
    version_id        bigint                                 not null
        constraint uq_version_id
            unique,
    text_version      text                                   not null,
    gar_xml_full_url  text,
    gar_xml_delta_url text,
    exp_date          timestamp,
    date              date,
    created_at        timestamp with time zone default now() not null,
    updated_at        timestamp with time zone default now() not null,
    status            text
);

create index idx_version_info_date
    on version_info (date);
