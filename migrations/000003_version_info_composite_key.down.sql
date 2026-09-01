alter table version_info drop constraint uq_version_id_file_type;
alter table version_info add constraint uq_version_id unique (version_id);
