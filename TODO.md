# Y10k TODOs

- [ ] Fix issue with 0-byte bzip2 files

- [ ] Reuse DB Tx's when adding packages to a yum.Repo

- [ ] Refactor funcs to isolate reposync and creatrepo

- [ ] Enable adding packages to existing databases

- [x] Enable transactions for batch imports in createrepo

- [x] Ensure SQLite3 inserts are syncronised to a single thread

- [x] Move bzip logic out of yum.DB and into yum.Repo