# ectocheck

Hunt the ghosts in an AI's answer before you trust it.

A "ghost" is a claim that cannot point to real evidence. Language models
invent them constantly: plausible-sounding facts that nothing ever said.
ectocheck takes an AI-generated answer and your local source files (docs,
logs, code, notes), and checks every claim against what your sources
actually contain. Claims that cannot show their evidence get marked, and
the exit code tells your pipeline the answer is not to be trusted.

Everything runs locally: a small open-weight model through
[Ollama](https://ollama.com) does the judging, and your files never leave
the machine.

## How it works

```
answer.md ──► split into claims ──► retrieve candidate evidence ──► local model judges ──► citations validated ──► report
                 (deterministic)      (token overlap, no magic)      (SUPPORTED/PARTIAL/GHOST)   (invented ones rejected)
```

1. **Claims.** The answer is split into atomic claims, deterministically:
   bullets and sentences, headings and boilerplate dropped.
2. **Retrieval.** Each claim is matched against evidence chunks from your
   sources by plain token overlap. Simple on purpose: you can see exactly
   why a chunk was picked, and the same input always picks the same chunks.
3. **Judgment.** A local model (default `qwen2.5:1.5b`) gets one claim and
   its top candidate chunks, and returns SUPPORTED, PARTIAL or GHOST with
   the evidence it relied on.
4. **The part that matters.** The judge's citations are checked too. It
   cites evidence positionally (`[1]`, `[2]`, ...), and any citation it was
   never shown is treated as invented: the verdict is thrown out and the
   claim comes back UNVERIFIED. A SUPPORTED with no valid citation is an
   assertion, not a verdict. ectocheck does not trust its own judge by
   default — because the judge is exactly the kind of model it exists to
   catch.

## Install

Requires Go 1.26+ and [Ollama](https://ollama.com) with the judge model:

```sh
ollama pull qwen2.5:1.5b
go install github.com/DhanushNehru/ectocheck@latest
```

Or build from source: clone, `go build -o ectocheck .`

## Usage

```sh
ollama serve   # if it isn't running
ectocheck check answer.md --src ./docs --src ./runbook.log
```

Flags: `--model` (default `qwen2.5:1.5b`), `--ollama` (server URL),
`--top-k` (evidence chunks per claim, default 3), `--format text|json`.

Exit code is `1` when any claim comes back GHOST or UNVERIFIED, so
`ectocheck check ... && deploy.sh` only deploys when every claim checked
out.

## Real sample output

The repo ships `examples/homeserver/` (notes and a backup log for a
fictional home server) and `examples/answer.md`, an "AI answer" about it
with two invented claims mixed in. Actual output of
`ectocheck check examples/answer.md --src examples/homeserver --model qwen2.5:1.5b`:

```
✓ SUPPORTED  The home server is a Beelink Mini S12 with an N100 CPU and 16 GB of RAM.
              The evidence [1] directly states that the home server is a Beelink Mini S12 with an N100 CPU and 16 GB of RAM, which matches the claim.
              [E2] examples/homeserver/README.md:3 — The box is a Beelink Mini S12 with an N100 CPU and 16 GB of RAM. It runs Debian 12 and liv…

? UNVERIFIED Nightly backups copy /opt/stacks to /mnt/usb/backups and verify the result.
              model claimed support but cited no valid evidence
              (note: the judge cited evidence it was never shown; verdict not trusted)

✓ SUPPORTED  Jellyfin serves media on port 8096.
              The evidence [1] confirms that Jellyfin services are run as docker compose stacks, which includes serving media on port 8096.
              [E3] examples/homeserver/README.md:6 — Services run as docker compose stacks in /opt/stacks. Jellyfin serves media on port 8096. …

? UNVERIFIED The server also runs Pi-hole for network-wide ad blocking on port 53.
              model claimed support but cited no valid evidence
              (note: the judge cited evidence it was never shown; verdict not trusted)

? UNVERIFIED Backups failed twice last week because the USB disk filled up.
              model claimed support but cited no valid evidence
              (note: the judge cited evidence it was never shown; verdict not trusted)

5 claims: 2 supported, 0 partial, 0 ghosts, 3 unverified
No ghosts, but not every claim could be trusted. Treat unverified claims as unchecked.
```

Both invented claims (the Pi-hole that never existed, the disk failure
that never happened) come back flagged and untrusted, and the exit code
is 1. Note the middle claim: a *true* claim also came back UNVERIFIED,
because the judge asserted support without producing a citation that
survives checking. That is the tool refusing to rubber-stamp, and this
sample is the real output, not a curated one.

## Honest limitations

Read these before relying on ectocheck for anything important.

- **A real citation is not proof.** The guard verifies that cited evidence
  exists and was actually shown to the judge. It does not and cannot prove
  the evidence really entails the claim. In testing, the small default
  judge once called an invented claim SUPPORTED and cited real-but-
  irrelevant evidence for it; citation validation cannot catch that.
  Treat SUPPORTED as "worth trusting more", never as verified fact.
- **Small judges are weak judges.** `qwen2.5:0.5b` often failed to follow
  the answer format (those claims land in UNVERIFIED) and once approved an
  invented claim outright. `qwen2.5:1.5b` is the smallest model that
  behaved acceptably in testing. Larger local models should judge better;
  none turn this into a guarantee.
- **Keyword retrieval misses things.** Synonyms and paraphrase between a
  claim and its evidence can score zero overlap, which pushes good claims
  toward GHOST. False alarms are a design choice: ectocheck errs toward
  not trusting.
- **Deterministic claim splitting is naive.** Headings and the DEV
  challenge boilerplate are skipped, but a sentence with several facts is
  judged as one claim.
- Linux/macOS. No GPU required, but bigger judges want the RAM.

## Why local, why open

The answers being checked often quote private things: logs, customer
notes, internal runbooks. A verification step that ships them to a hosted
API defeats the point. With an open-weight judge running on the same
machine, the answer and the evidence stay put, the whole pipeline works
on a laptop with the internet off, and you can swap the judge for any
model Ollama runs.

## License

MIT
