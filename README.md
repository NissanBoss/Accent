# Accent

What ties your anonymous code to you.

```
    [NAMES  ] lineup and outtakes name the same owner
              copyright holder     N********s
                                   LICENSE:3  and  LICENSE:3

    [SHARED ] lineup and outtakes share 64 pieces of text nobody else has
              a comment, report.go:3 and report.go:3
                  W**t a f*****g i*, a*d h*w i* r*****s t*e s****n.
              9 lines, build.sh:64 and build.sh:64
                  # z*p f***t: i* i* w**t L***x a*d t*e w******w h**e. p*...
              a message, report.go:61 and report.go:104
                  N*****g w*s c*****d, a*d n*****g w*s s**t a******e.
              ... and 61 more

    [WORDS  ] The Go in lineup and outtakes uses the same rare words as one person
              would: closer than any two strangers in the corpus
              1,123 words few people use, rarest first:
                a name       F************************x
                a name       T*****************e
                a name       R**************g
              ... and 1120 more
```

That is two other tools of mine, compared: of course they are tied. The
licence names me, the build script was copied from one to the other, and
the code is written in the same words.

## The problem

People publish code under names that are not theirs for good reasons. The
developer of a tool their government would rather did not exist: in August
2015 the author of [Shadowsocks](https://en.wikipedia.org/wiki/Shadowsocks),
who published under a handle, wrote that the police had come and asked them
to delete the code, and they did. The security researcher who does not want
an exploit on their CV. The person whose employer owns everything they write
under their own name.

They tend to look after the obvious things: the git email, the account, the
times of day. What they publish is still code, and code has an accent. The
helper copied from the old project into the new one, the comment that
explains the same workaround in the same words, the error message, the
build script. The verbs somebody reaches for in every name they write,
`load` where somebody else writes `read`, `cfg` where somebody else writes
`conf`. The module path in a forgotten `go.mod`, the address in a test, the
Ko-fi page in a funding file.

Each of these used to need a patient reader. A code search finds a copied
line in seconds, and a language model reads the rest. In 2025 [Choi and
colleagues](https://arxiv.org/abs/2501.08165) had language models attribute
code to its author among 686 people from a single example of each, right
nearly seven times in ten.

## What already exists

The research is on the attacking side. [Caliskan and
colleagues](https://www.usenix.org/conference/usenixsecurity15/technical-sessions/presentation/caliskan-islam)
picked one programmer out of 1,600 from their code 94 times in 100 in 2015,
on Google Code Jam answers, with a trained classifier. The 2025 paper above
did it with prompts. [GitFive](https://github.com/mxrch/GitFive) digs the
email addresses and other accounts out of a GitHub profile.

On the defending side there is [Stylog](https://github.com/GH05TCREW/stylog),
a stylometry toolkit for prose and code whose verdict on two samples needs a
verifier fitted on labelled examples first, and
[Style Counsel](https://doi.org/10.1145/3267323.3268951), a 2018 paper whose
tool reads a trained classifier to tell a programmer what to change for
their code to pass as somebody else's. [gitleaks](https://github.com/gitleaks/gitleaks)
finds secrets, which is a different question: a copied comment is not a
secret, and it ties two repositories together anyway.

None of them takes two folders and says what ties them: which names, which
lines, which words, so the owner can change them before publishing. Accent
does, with nothing to train and no model, and its verdicts are set against
what real people's projects actually share.

## How it reads

**What was not written by hand is left out.** Vendored libraries,
`node_modules`, generated code, minified bundles, copies of jQuery and
friends, lock files. Measuring them would measure somebody else, and two
projects that vendor the same library would look like one author. The
report says how much was left out.

**Names** come first: the email addresses, the account in a module path or
a manifest's repository field, the copyright holder, the funding accounts,
analytics and ad IDs, a Sentry or Firebase or Supabase project, a Docker
image under a username, a GitHub Pages site, a wallet address. Two projects
that carry the same one belong to the same account, and nothing about it is
a matter of judgement. Names that turned up in three or more different
people's projects in the calibration corpus, the `grafana` and `prom` images
on Docker Hub say, are nobody's and are left out.

**Shared text** is next: three lines or more in a row that both projects
have, and comments and messages with the same words. Each one is a search
away from the other project. Lines that three or more different people in
the corpus have are left out, and so are imports, Android layouts, build
manifests, licences and ignore files, which strangers share all the time.

**Words** are last, and are where the calibration ended up. The words of
each project's names, comments and messages are collected, `parseRetryAfter`
as `parse`, `retry` and `after` and as the whole name, and every word both
projects use counts by how rare it is: nothing if more than one project in
ten in the corpus uses it, more the fewer do. The answer is set against what
pairs of strangers in the corpus share: "closer than 95 pairs of strangers
in 100".

**On one folder** it says what in it names its owner, and which of its
habits sit two standard deviations or more from how the people in the
corpus write the same language: a name used five times as often as most
people use it, the small words its comments lean on, how much of it is
comments at all.

The report masks names, shared lines and words unless you pass `--reveal`,
so it can be pasted somewhere as it is. Secrets are covered either way.

## The calibration

**The corpus.** 784 repositories by 313 people on GitHub, two or three each
for all but thirteen of them, and every one written alone: the contributor
list is the owner and nobody else. Go, Python, JavaScript, TypeScript, Rust
and Java, last pushed before June 2022 so that coding assistants had not
written much of it. They are other people's code, so they are not in this
repository, and only counts were taken. The tables in `data/` that came out
of them hold counts and hashes, never a line anybody wrote or anybody's
name.

The question asked is the one that matters: two different projects by the
same person, against two projects by strangers. Every number below was
measured on one half of the people, with what counts as common or rare
taken from the other half, the way a project of yours is measured against
a corpus it is not in. A pair needs two projects of 3,000 tokens or more,
which 158 of the people had.

| tied together by | same person, 158 pairs | strangers, 402 pairs |
|---|---|---|
| a name both carry | 50.6% | 0.2% |
| shared text: three lines, a comment or a message | 39.9% | 2.0% |
| words, closer than 99 strangers in 100 | 31.6% | 0.5% |
| any of them | 69.0% | 2.2% |
| any of them but names | 45.6% | 2.2% |

The last row is the one for somebody who has already taken their name out:
nearly half of these people's pairs of projects are still tied together by
what they copied and how they write.

The words on their own, by language, as the chance that a pair by one
person comes out closer than a pair by two:

| Go | Python | JavaScript | TypeScript | Rust | Java |
|---|---|---|---|---|---|
| 0.86 | 0.88 | 0.77 | 0.75 | 0.76 | 0.80 |

Half of the same person's pairs come out closer than 95 pairs of strangers
in 100, and nearly a third closer than 99 in 100. Only the second is called
a tie. At 95 one pair of strangers in twenty would be called tied too, so
there the report lists the words as a hint. The tables that ship were
measured the same way, each person against the corpus without them, and
came out the same: a third of one person's pairs past the line for a tie,
and one pair of strangers in a hundred.

**What the calibration changed.** Almost everything:

- The first measure was Burrows's Delta over how often each keyword,
  bracket and kind of name turns up, the method
  [Tell](https://github.com/NissanBoss/Tell) uses on prose. On real people's
  repositories it barely told one person's two projects from two strangers',
  0.64 to 0.73 by language, because what a project is about moves those
  counts far more than who wrote it. Scaling each feature by how much
  people differ on it helped in four languages and hurt in two. Rare shared
  words beat both in all six, and adding Delta to the words made them worse
  in four languages and no better in the other two.
- A single shared line turned up between strangers' projects one time in
  five. Two together, one time in eighteen. Three, one time in eighty,
  mostly where both had pasted in the same library or the same tutorial. So
  a line or two is listed and does not count.
- The first strangers line was drawn with every pair scored against the
  whole corpus, their own projects in it, so each word two strangers shared
  counted as more common than it was. Measured the way a project from
  outside is measured, three pairs of strangers in a hundred crossed the
  line meant for one, and fifteen the line meant for five. Each pair is now
  scored against the corpus without either person, and the line holds: one
  in a hundred, and five.
- Android layouts, `pom.xml` and `go.mod` were behind much of what strangers
  shared, and are left out of the shared text.
- A README's links to other people's projects were being read as the
  owner's accounts. Only the module line, a manifest's repository field, the
  funding file and the clone line count now.
- On one folder, every small word in a long commenter's comments came out
  as used ten times as often as anybody's, which said one thing ten times.
  Each habit is now measured within its kind, comments against comments,
  names against names, and how much of each kind there is goes in once.

## Running it

```bash
accent named-project/ anonymous-project/
```

```bash
accent project/
```

```bash
accent a/ b/ c/
```

Every pair of folders given is compared. `--reveal` prints the names, the
shared lines and the words; secrets are covered either way.

Exit code is 1 when two folders are tied together by a name, by shared text
or by their words, 3 when no code could be read, 0 otherwise. So it can be
the last thing that runs before the anonymous project is pushed, with the
named one next to it.

Windows marks anything that came from the internet, so the first run may
bring up a blue "Windows protected your PC" box. Right click the zip,
Properties, tick Unblock at the bottom, and then extract it.

## What it cannot do

**It does not prove anything.** A tie is evidence that a person comparing
the two projects would stop at. Two strangers came out tied 2.2% of the
time, mostly because both had pasted in the same library or tutorial. And
no tie is not innocence: nearly a third of the same person's pairs of
projects showed nothing, and a reader with more patience, or a model, may
find what this did not.

**It reads code, not history.** The commit times and the author lines are
what [Alibi](https://github.com/NissanBoss/Alibi) reads, and what a
repository gives away on being made public, deleted files included, is
[Airing](https://github.com/NissanBoss/Airing). How you write prose, in a
README or an issue, is [Tell](https://github.com/NissanBoss/Tell).

**Words need code.** About 3,000 tokens of one language on each side, or
the words are not compared. A small script says little, which is fine: a
small script also gives little away.

**Six languages were calibrated.** Go, Python, JavaScript, TypeScript, Rust
and Java have their own tables. C, C++, C#, Kotlin, Swift, Ruby, PHP, shell
and Lua are read and compared against all six together, which is rougher,
and on one folder their habits are not described at all.

**The corpus is from before coding assistants.** Code an assistant wrote
uses its words, not yours, and that is likely to make a project harder to
tie to anybody, its author included. It has not been measured here.

**Names are found by pattern.** An address, a module path, an ID in a known
format. A name written in a way no pattern expects, a nickname in a comment,
is not found.

**It does not fix anything.** It says what to change. Changing it is yours,
and for an anonymous project already published, it is worth assuming
everything in it has been read.

## It only reads

Nothing is written, nothing is sent anywhere, no other program is run, and
there is no model in it. Tests read the source and fail the build if a call
that writes a file, an import of anything under `net`, an import of
`os/exec` or anything that looks like a model turns up.

That matters more here than anywhere. The folder you point this at is the
one you are trying to keep apart from your name, and checking whether it can
be traced by sending it to somebody else's service would be the leak
itself. It is also why the method is arithmetic: every number in the report
is a count, a share or a table in `data/`, and can be checked by hand.

The tables are counts from the corpus, and they are the only thing in the
program that came from other people's code. The rare words, the lines and
the names in them are hashes. What is in the clear is what a fifth or more
of the people writing a language use, names like `err` and `request`, and
the libraries three or more of them use. The only code that writes the
tables is a test, which never ships in the binary.

## Building

Needs [Go](https://go.dev) and nothing else. No dependencies at all.

```bash
go build -o accent .
```

```bash
go test ./...
```

## Licence

MIT. Do what you like with it.
