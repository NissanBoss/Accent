# Changelog

## v1

The first release.

Says what ties an anonymous project to the named one its author also
wrote.

- **Names first**: email addresses, the account in a module path or a
  manifest, the copyright holder, funding accounts, analytics and ad IDs,
  Sentry, Firebase and Supabase projects, hosted apps, Docker images under a
  username, wallets and PGP keys. Names three or more people in the corpus
  carry are nobody's and are left out.
- **Shared text**: three lines or more in a row, and comments and messages in
  the same words, that nobody else in the corpus has. Imports, licences,
  build manifests and Android layouts are left out, because strangers share
  them all the time.
- **Words**: the rare words of names, comments and messages that both
  projects use, counted by how few people use them and set against what two
  strangers writing the same language share.
- **One folder** on its own: what in it names its owner, and the habits that
  set it apart from how the corpus writes its language.
- **Fifteen languages** read: Go, Python, JavaScript, TypeScript, Rust, Java,
  C, C++, C#, Kotlin, Swift, Ruby, PHP, shell and Lua. Vendored, generated
  and minified code and copies of well known libraries are left out, and
  counted.
- **Masked by default**, and secrets covered whether `--reveal` is passed or
  not.

**Calibrated on 784 repositories by 313 people** who each wrote theirs alone,
before coding assistants. Two projects by the same person came out tied
69.0% of the time, and two by strangers 2.2%. With the names taken out, the
lines they copied and the words they use still tied 45.6% of the same
person's pairs together. The code is not in the repository, and the tables
built from it hold counts and hashes.

Nothing is written, nothing is sent anywhere, no other program is run, and
there is no model in it. Tests read the source and fail the build if any of
that changes.

Builds for Windows, Linux and macOS, on Intel and ARM. No dependencies.
