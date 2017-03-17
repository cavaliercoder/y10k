# Y10k TODOs

- [ ] Test primary_db with actualy yum client

- [ ] Refactor funcs to isolate reposync and creatrepo

- [ ] Enable adding packages to existing databases

- [ ] Implement db_info.checksum in sqlite dbs

- [x] Reuse DB Tx's when adding packages to a yum.Repo

- [x] Fix issue with 0-byte bzip2 files

- [x] Enable transactions for batch imports in createrepo

- [x] Ensure SQLite3 inserts are syncronised to a single thread

- [x] Move bzip logic out of yum.DB and into yum.Repo
