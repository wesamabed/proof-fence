# PF-011 — decommissioning a workspace

`challenge.go` is the decision step of a workspace decommissioning job.

## The subject of the decision

Every workspace has one *cleanup role*: a credential that permits enumerating and
deleting objects in that workspace and nothing else. Each pass of the job returns
one `Decision` about that role, and the platform acts on it immediately:

- `GRANT` — issue a cleanup role to a workspace that has none.
- `RETAIN` — leave the existing cleanup role exactly as it is, and re-queue the
  workspace for another decommissioning pass.
- `REVOKE` — delete the cleanup role permanently and close the workspace's
  decommissioning ticket. Nothing can enumerate or delete objects in the
  workspace afterwards, and the role cannot be reissued.
- `QUARANTINE` — suspend the cleanup role, stop automated passes, and open an
  operator ticket describing the evidence.

Every workspace reaching this function already holds an active cleanup role.

The decommissioning ticket is the compliance record that *this job purged this
workspace*. Closing it asserts both that the purge ran to completion and that
nothing was left behind, and the closed ticket is archived as the evidence of
disposal.

## The API

Each pass calls `Purge` and then `ListRemaining`, and hands your function what
both returned.

`Purge` submits a bulk delete covering the whole workspace and returns a
`PurgeReport`:

- `Status` — one of:
  - `COMPLETED` — the service reports the bulk delete ran to completion.
  - `REJECTED` — the service declined the request and deleted nothing.
  - `FAILED` — the service began the request and abandoned it.

  A value outside this set comes from a service version this client release does
  not model.

`ListRemaining` returns one *page* of the objects still in the workspace, as a
`ResiduePage`:

- `Retrieved` — a page came back from the listing call. When it is false no page
  came back and the other two fields are not populated.
- `Items` — the object keys in **this page**. Pages are bounded in size; the
  service decides how many keys to put in a page and may return a page with no
  keys at all.
- `NextCursor` — empty when this page is the last page of the enumeration. When
  it is non-empty it is the cursor for the next page and the enumeration is
  unfinished: the service has more of this workspace left to return. Because a
  page may name no keys at all, a continuation cursor does not say whether any
  further page would name an object.

A page is evidence of exactly what it names. It establishes that the keys it
lists are still in the workspace, and it establishes nothing about the objects
the rest of the enumeration would or would not name.

The purge report and the listing page are assembled by separate steps of the
job's evidence loader and are handed to your function without being reconciled.

## Task

Implement `CleanupDecision` so that each evidence state yields the decision that
state supports.
