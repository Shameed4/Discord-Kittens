#!/usr/bin/env bash
#
# the proof run: stand up the 3-node cluster, throw a swarm of bots at it,
# kill a node while games are in flight, and see how many games survived.
# target is 100% — a node dying should lose zero games.
#
# the number that matters is the bot swarm's own "games survived" line. the
# bots don't know anything about redis or snapshots; they just watch whether
# their game keeps talking to them after the kill.
#
# usage:  scripts/chaos.sh   (tweak with BOTS/PER_LOBBY/DURATION_S/KILL_AT/KILL_NODE)
set -euo pipefail

# tunables
TARGET="${TARGET:-ws://localhost:8080}"
BOTS="${BOTS:-60}"
PER_LOBBY="${PER_LOBBY:-4}"
DURATION_S="${DURATION_S:-120}"   # total run length, seconds
KILL_AT="${KILL_AT:-40}"          # seconds into the run to kill a node
KILL_NODE="${KILL_NODE:-node2}"

# make sure bot has enough time to register dead lobby
ORPHAN_TIMEOUT=30
MIN_TAIL=$((ORPHAN_TIMEOUT + 10))
if (( DURATION_S - KILL_AT < MIN_TAIL )); then
  echo "not enough runway after the kill (need >= ${MIN_TAIL}s, got $((DURATION_S - KILL_AT))s)" >&2
  echo "bump DURATION_S or drop KILL_AT" >&2
  exit 1
fi

cd "$(dirname "$0")/.."            # repo root
REPORT="$(mktemp -t chaos-report.XXXXXX)"

cleanup() { echo "--- tearing down cluster ---"; docker compose down -v --remove-orphans >/dev/null 2>&1 || true; rm -f "$REPORT"; }
trap cleanup EXIT

# --- 1. bring up the cluster ------------------------------------------------
echo "--- building + starting 3-node cluster ---"
docker compose up --build -d

# don't start the bots until nginx can actually reach a live node
echo "--- waiting for the cluster to come up on :8080 ---"
for i in $(seq 1 60); do
  if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then break; fi
  if (( i == 60 )); then echo "cluster never came healthy" >&2; exit 1; fi
  sleep 1
done
echo "cluster healthy"

# --- 2. launch the loadbot swarm --------------------------------------------
echo "--- launching ${BOTS} bots (${PER_LOBBY}/lobby) for ${DURATION_S}s against ${TARGET} ---"
( cd game && go run ./loadbot \
    -target "$TARGET" \
    -bots "$BOTS" \
    -perLobby "$PER_LOBBY" \
    -duration "${DURATION_S}s" ) >"$REPORT" 2>&1 &
LOADBOT_PID=$!

# --- 3. pull the rug: kill a node while games are in flight ------------------
sleep "$KILL_AT"
echo "--- killing ${KILL_NODE} at t+${KILL_AT}s ---"
docker compose kill "$KILL_NODE"

# --- 4. let it ride, then see who made it -----------------------------------
echo "--- letting the swarm play on for another ~$((DURATION_S - KILL_AT))s ---"
wait "$LOADBOT_PID"

# the other side of the story: each surviving node bumps this counter every
# time it picks up a lobby the dead node dropped. handy sanity check that the
# games survived because they were actually adopted, not just by luck.
echo
echo "=== lobbies adopted by survivors (kittens_failovers_total) ==="
total_failovers=0
for n in node1 node2 node3; do
  [ "$n" = "$KILL_NODE" ] && continue
  v=$(docker compose exec -T "$n" wget -qO- http://localhost:8080/metrics 2>/dev/null \
        | awk '/^kittens_failovers_total /{print $2}') || v=""
  v=${v:-0}
  printf "  %-6s adopted %s lobbies\n" "$n" "${v%.*}"
  total_failovers=$((total_failovers + ${v%.*}))
done
echo "  total adopted: ${total_failovers}"

echo
echo "=== loadbot report ==="
cat "$REPORT"

# pull the one line everyone actually cares about back up to the bottom
echo
grep "games survived" "$REPORT" || true
if grep -q "games survived: .*100\.0%" "$REPORT"; then
  echo "RESULT: PASS — zero games lost to the node kill"
else
  echo "RESULT: investigate — survival below 100% (see report above)"
fi
