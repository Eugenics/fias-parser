alter table version_info drop constraint uq_version_id;
alter table version_info add constraint uq_version_id_file_type unique (version_id, file_type);
