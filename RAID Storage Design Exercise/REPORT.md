# RAID Storage Design Exercise

## 1. Scenario Requirements

| Requirement | Value |
|---|---|
| **Total Usable Capacity Needed** | 12 TB |
| **Redundancy Level Desired** | Must withstand the failure of **two** disks |
| **Performance Needs** | High **read** speed, suitable for data analytics workloads |
| **Assumed Drive Size** | 2 TB (HDD), used consistently across configurations for a fair capacity/cost comparison |

## 2. Research & Simulation

RAID levels considered, per the assignment prompt: **RAID 0, RAID 1, RAID 5, RAID 6, RAID 10**.

Each candidate was configured in the [Seagate RAID Capacity Calculator](https://www.seagate.com/products/nas-drives/raid-calculator/) to find the drive count needed to reach (or approach) **12 TB usable capacity**, holding drive size at 2 TB wherever the calculator allowed a clean, round configuration. RAID 10 required 3 TB drives instead of 2 TB to reach exactly 12 TB usable without wasted/unused capacity — noted below.

- **RAID 0** — striping only, no parity or mirroring. Maximizes capacity and performance, but has **zero fault tolerance**: a single drive failure destroys the entire array. Included only as a performance/cost baseline, since it cannot satisfy the "withstand two disk failures" requirement at all.
- **RAID 1** — mirroring. Every drive has a full duplicate. Protects against drive failure but is extremely capacity-inefficient (50% overhead for only single-failure tolerance per mirrored pair), and doesn't scale efficiently to 12 TB usable with 2 TB drives.
- **RAID 5** — single distributed parity. Tolerates exactly **one** drive failure. Good capacity efficiency, but does not meet the stated 2-disk-failure requirement.
- **RAID 6** — dual distributed parity. Tolerates **any two** drive failures simultaneously, with good capacity efficiency. Requires a minimum of 4 drives.
- **RAID 10** — striped mirrors (RAID 1+0). Can tolerate multiple drive failures, but only if the failed drives are **not from the same mirrored pair** — i.e., 2-disk fault tolerance is *probabilistic*, not guaranteed, unlike RAID 6.

## 3. Data Collection

All values below were produced directly by the Seagate RAID Capacity Calculator (drive size noted per row).

| RAID Level | Drive Size | Drives Required | Total Raw Capacity | Usable Capacity | Unused | Fault Tolerance | Read/Write Performance Characteristics |
|---|---|---|---|---|---|---|---|
| **RAID 0** | 2 TB | 6 | 12 TB | 12.00 TB | 0 TB | **None** — any single drive failure causes total data loss | Best possible read **and** write performance (full striping across all drives); performance scales linearly with drive count |
| **RAID 1** | 2 TB | 2 (mirrored pair, capped at 2 TB usable) | 4 TB | 2.00 TB usable per pair | — | Tolerates 1 failure per mirrored pair | Strong read performance (can read from either mirror); write performance ≈ single drive (writes duplicated, not parallelized); impractical to reach 12 TB usable without stacking many pairs (6 pairs = 12 drives for 12 TB) |
| **RAID 5** | 2 TB | 7 | 14 TB | 12.00 TB | 0 TB | Tolerates **1** drive failure only | Good read performance (data striped across n−1 drives); write performance reduced by parity read-modify-write overhead; performance degrades significantly during a rebuild |
| **RAID 6** | 2 TB | 8 | 16 TB | 12.00 TB | 0 TB | Tolerates **any 2** drive failures | Read performance similar to RAID 5; write performance lower than RAID 5 (dual-parity calculation overhead on every write); rebuild is slower than RAID 5 due to larger arrays typically paired with RAID 6, but the array stays protected if a second drive fails mid-rebuild |
| **RAID 10** | 3 TB* | 8 | 24 TB | 12.00 TB | 0 TB | Tolerates multiple failures **if not in the same mirrored pair**; a worst-case pair failure still causes data loss | Excellent read **and** write performance — data is striped (RAID 0 layer) across mirrored pairs (RAID 1 layer), avoiding the parity-calculation penalty of RAID 5/6 entirely; best-performing redundant option for high-IOPS/analytics workloads |

*\*Note on RAID 10 and 2 TB drives: the Seagate calculator caps a single RAID 10 array at 8 active drives — additional 2 TB drives beyond 8 were placed into the array but reported as "unused" capacity (e.g., 12×2TB drives reported only 8.00 TB usable / 8.00 TB protected / 8.00 TB unused). To reach a clean 12 TB usable figure in RAID 10 within that 8-drive practical limit, 3 TB drives were substituted (8 × 3 TB = 24 TB raw → 12 TB usable, 12 TB mirrored, 0 TB unused).*

## 4. Analysis

### Capacity vs. Redundancy Trade-offs
- **RAID 0** gives 100% capacity efficiency but 0% redundancy — the two are at complete odds, and it is disqualified by the assignment's own redundancy requirement.
- **RAID 5** gives high capacity efficiency (n−1 of n drives usable: 6/7 ≈ 86% efficient here) but only single-drive protection — a second failure during a rebuild is catastrophic.
- **RAID 6** trades a bit more capacity for drive count (n−2 of n drives usable: 6/8 = 75% efficient here) in exchange for *guaranteed* two-drive fault tolerance, which is exactly what this scenario requires.
- **RAID 10** gives the strongest *effective* fault tolerance per mirrored pair but at the steepest capacity cost (50% efficiency, always) — reaching 12 TB usable required the most raw capacity (24 TB) of any option, and its 2-disk tolerance is conditional, not guaranteed.

### Performance Considerations
- Both **RAID 6** and **RAID 5** incur a write penalty from parity calculation; RAID 6 is worse than RAID 5 here because every write touches two parity blocks instead of one. For a predominantly **read**-heavy analytics workload (as specified), this write penalty matters far less than it would for a transactional/write-heavy workload.
- **RAID 10** has no parity penalty at all — writes go straight to a mirror pair — giving it the best all-around performance, but that performance advantage is most valuable for write-heavy or latency-sensitive workloads (e.g., OLTP databases), not pure read-heavy analytics.
- For a read-heavy analytics workload specifically, RAID 5 and RAID 6 both deliver strong sequential/parallel read throughput (reads are striped across all data drives), making RAID 6's write penalty largely irrelevant to this use case.

### Cost Implications
- **RAID 6** needs 8×2TB drives (16 TB raw) to deliver 12 TB usable — 4 TB (2 drives' worth) spent on redundancy overhead.
- **RAID 10** needs 8×3TB drives (24 TB raw) to deliver the same 12 TB usable — 12 TB (4 drives' worth, and notably larger/costlier drives) spent on redundancy overhead, roughly double the raw capacity (and real dollar cost) of RAID 6 for the identical usable capacity.
- **RAID 5** is the cheapest option (7×2TB = 14 TB raw for 12 TB usable, only 2 TB overhead) but fails the redundancy requirement outright, so its lower cost cannot be credited as a valid option here.

## 5. Recommendation

**Recommended configuration: RAID 6, 8 × 2 TB drives.**

- **Meets the capacity requirement:** 8×2TB drives (16 TB raw) deliver exactly 12.00 TB of usable capacity, matching the requirement with no wasted/unused space.
- **Meets the redundancy requirement:** RAID 6's dual distributed parity *guarantees* survival of **any** two simultaneous drive failures — unlike RAID 10, which only tolerates two failures if they happen to avoid the same mirrored pair. Since the requirement explicitly says "must withstand the failure of two disks" (not "some" failures of two disks), RAID 6 is the only option considered that provides a deterministic guarantee.
- **Aligns with performance expectations:** the workload is explicitly read-heavy analytics, not write-heavy transactional. RAID 6's main weakness — write performance, due to dual-parity overhead — is largely irrelevant here, while its read performance (striped across all data drives) is strong.
- **Compromises/considerations:** RAID 6 does cost 4 TB of raw capacity in overhead (2 drives' worth) and will have slower writes and a longer rebuild window than RAID 10 if a drive does fail. If this system's workload profile changes to include significant write/transactional traffic, RAID 10 (accepting its higher drive cost and probabilistic rather than guaranteed 2-disk tolerance) would be worth revisiting. RAID 5 and RAID 0 are both disqualified outright by the explicit two-disk-failure requirement.

---
*Data sourced from the [Seagate RAID Capacity Calculator](https://www.seagate.com/products/nas-drives/raid-calculator/).*
