create type check_suite_status as enum('pending', 'dispatched');

create table check_suites (
  id serial primary key,
  github_id integer not null,
  status check_suite_status not null default 'pending',
  github_installation_id integer not null,
  repo_name text not null,
  repo_owner text not null,
  repo_clone_url text not null,
  commit_ref text not null,
  created timestamp with time zone not null default current_timestamp
);

create or replace function tf_check_suites()
  returns trigger AS
$BODY$
  begin
if TG_OP = 'INSERT' then
  PERFORM pg_notify('new_check_suites', new.id::text);
end if;
return null;
end; $BODY$
  language plpgsql volatile security definer;

create trigger t_check_suites
  after insert
  on check_suites
  for each row
  execute procedure tf_check_suites();

create type job_status as enum('queued', 'in_progress', 'skipped', 'success', 'failure');

create table jobs (
  id serial primary key,
  check_suite integer not null references check_suites(id) on delete cascade,
  github_check_run_id integer,
  status job_status not null default 'queued',
  github_status job_status,
  runner text,
  skipped_by_runners text[] not null default '{}',
  test_requirements jsonb not null,
  test_results jsonb not null default '[]',
  ts_start timestamp with time zone,
  ts_end timestamp with time zone
);

create or replace function tf_jobs()
  returns trigger AS
$BODY$
  begin
if TG_OP = 'INSERT' then
  PERFORM pg_notify('new_jobs', new.id::text);
elsif TG_OP = 'UPDATE' then
  PERFORM pg_notify('changed_jobs', new.id::text);
end if;
return null;
end; $BODY$
  language plpgsql volatile security definer;

create trigger t_jobs
  after insert or update
  on jobs
  for each row
  execute procedure tf_jobs();