create table if not exists roles
(
    id   serial
        primary key,
    name varchar(255) not null
        unique
);

alter table roles
    owner to postgres;

create table if not exists users
(
    id       uuid default gen_random_uuid() not null
        primary key,
    username varchar(255)                   not null
        unique,
    password varchar(96)                    not null,
    email    varchar(255)                   not null
        unique
);

alter table users
    owner to postgres;

create table if not exists user_roles
(
    user_id uuid    not null
        references users
            on delete cascade,
    role_id integer not null
        references roles
            on delete cascade,
    primary key (user_id, role_id)
);

alter table user_roles
    owner to postgres;

create table if not exists refresh_tokens
(
    user_id      uuid                                                                     not null
        references users
            on delete cascade,
    hashed_token varchar(512)                                                             not null
        unique,
    created_at   timestamp with time zone default CURRENT_TIMESTAMP                       not null,
    expires_at   timestamp with time zone default (CURRENT_TIMESTAMP + '1 day'::interval) not null,
    primary key (user_id, hashed_token)
);

alter table refresh_tokens
    owner to postgres;

create table if not exists courses
(
    id              uuid                     default gen_random_uuid() not null
        primary key,
    title           text                                               not null,
    description     text                                               not null,
    logo_object_key text,
    created_at      timestamp with time zone default now()             not null,
    updated_at      timestamp with time zone default now()             not null,
    author_id       uuid                                               not null
        references users
            on delete cascade,
    status          text                                               not null,
    stars_count     integer                  default 0                 not null
);

alter table courses
    owner to postgres;

create table if not exists lessons
(
    id           uuid                     default gen_random_uuid() not null
        primary key,
    course_id    uuid                                               not null
        references courses
            on delete cascade,
    lesson_title text                                               not null,
    lesson_order integer                                            not null,
    created_at   timestamp with time zone default now()             not null,
    updated_at   timestamp with time zone default now()             not null,
    module_id    uuid                                               not null,
    constraint unique_module_lesson_order
        unique (module_id, lesson_order)
);

alter table lessons
    owner to postgres;

create table if not exists contents
(
    id         uuid                     default gen_random_uuid() not null
        primary key,
    lesson_id  uuid                                               not null
        references lessons
            on delete cascade,
    type       text                                               not null
        constraint contents_type_check
            check (type = ANY (ARRAY ['text'::text, 'image'::text, 'video'::text, 'quiz'::text])),
    order_num  integer                                            not null,
    text       text,
    object_key text,
    quiz_json  jsonb,
    created_at timestamp with time zone default now()             not null,
    updated_at timestamp with time zone default now()             not null,
    unique (lesson_id, order_num)
);

alter table contents
    owner to postgres;

create table if not exists lesson_progress
(
    user_id    uuid                                   not null
        references users
            on delete cascade,
    lesson_id  uuid                                   not null
        references lessons
            on delete cascade,
    status     text                                   not null
        constraint lesson_progress_status_check
            check (status = ANY (ARRAY ['passed'::text, 'failed'::text])),
    score      double precision                       not null default 0,
    updated_at timestamp with time zone default now() not null,
    primary key (user_id, lesson_id)
);

alter table lesson_progress
    owner to postgres;

create table if not exists modules
(
    id           uuid                     default gen_random_uuid() not null
        primary key,
    course_id    uuid                                               not null
        references courses
            on delete cascade,
    title        text                                               not null,
    module_order integer                                            not null,
    created_at   timestamp with time zone default now()             not null,
    updated_at   timestamp with time zone default now()             not null,
    unique (course_id, module_order)
);

alter table modules
    owner to postgres;

create table if not exists course_subscriptions
(
    id         uuid                     default gen_random_uuid() not null
        primary key,
    course_id  uuid                                               not null
        references courses
            on delete cascade,
    user_id    uuid                                               not null
        references users
            on delete cascade,
    created_at timestamp with time zone default now()             not null,
    unique (course_id, user_id)
);

alter table course_subscriptions
    owner to postgres;

create table if not exists course_ratings
(
    course_id uuid not null
        references courses
            on delete cascade,
    user_id   uuid not null
        references users
            on delete cascade,
    primary key (course_id, user_id)
);

alter table course_ratings
    owner to postgres;

