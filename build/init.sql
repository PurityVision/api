-- Setup purity schema.
CREATE TABLE public.users
(
    uid bigint NOT NULL,
    email text NOT NULL,
    password text NOT NULL,
    PRIMARY KEY (uid)
);

ALTER TABLE public.users
    OWNER to postgres;

ALTER TABLE public.users
    ALTER COLUMN uid ADD GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 );

CREATE TABLE public.image_annotations
(
    hash text NOT NULL, -- Hash of the b64 content encoding.
    uri text NOT NULL,
    error text,
    date_added timestamp NOT NULL default CURRENT_TIMESTAMP,

    -- properties from Google's SafeSearchAnnotation type 
    adult smallint default 0,
    spoof smallint default 0,
    medical smallint default 0,
    violence smallint default 0,
    racy smallint default 0,
    -- end properties

    PRIMARY KEY (hash, uri)
);

ALTER TABLE public.image_annotations
    OWNER to postgres;

create table public.licenses
(
    id text,
    email text DEFAULT 'test@purityvision.com',
    stripe_id text,
    is_valid boolean default false,
    primary key (id)
);

-- Create the test database.
CREATE DATABASE purity_test WITH TEMPLATE purity;
