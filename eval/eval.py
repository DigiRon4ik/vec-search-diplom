"""
Evaluation script for semantic search quality and latency.

Usage:
    python eval/eval.py [--gateway http://localhost:8080] [--k 10]

Measures:
    - precision@k: fraction of top-k results whose category matches expected
    - latency: p50, p95, mean per query
"""

import argparse
import json
import statistics
import sys
import time
import urllib.request
import urllib.error


def search(gateway_url: str, query: str, k: int) -> tuple[list[dict], float]:
    url = f"{gateway_url}/api/search"
    payload = json.dumps({"query": query, "k": k}).encode()
    req = urllib.request.Request(url, data=payload, headers={"Content-Type": "application/json"})

    start = time.perf_counter()
    with urllib.request.urlopen(req, timeout=30) as resp:
        data = json.loads(resp.read())
    elapsed_ms = (time.perf_counter() - start) * 1000

    return data.get("results", []), elapsed_ms


def precision_at_k(results: list[dict], expected_categories: set[str]) -> float:
    if not results:
        return 0.0
    hits = sum(
        1 for r in results
        if r.get("product", {}).get("category", "") in expected_categories
    )
    return hits / len(results)


def main():
    parser = argparse.ArgumentParser(description="Evaluate semantic search")
    parser.add_argument("--gateway", default="http://localhost:8080", help="Gateway URL")
    parser.add_argument("--k", type=int, default=5, help="Number of results")
    parser.add_argument("--queries", default="eval/queries.json", help="Path to queries file")
    args = parser.parse_args()

    with open(args.queries, encoding="utf-8") as f:
        queries = json.load(f)

    print(f"Evaluating {len(queries)} queries against {args.gateway} (k={args.k})")
    print("=" * 80)

    precisions = []
    latencies = []
    details = []

    for i, q in enumerate(queries):
        query_text = q["query"]
        expected = set(q["expected_categories"])

        try:
            results, latency_ms = search(args.gateway, query_text, args.k)
        except (urllib.error.URLError, Exception) as e:
            print(f"  [{i+1}] FAIL: {query_text[:50]}... -> {e}")
            continue

        prec = precision_at_k(results, expected)
        precisions.append(prec)
        latencies.append(latency_ms)

        top_cats = [r.get("product", {}).get("category", "?") for r in results[:3]]
        status = "OK" if prec >= 0.5 else "LOW"
        details.append({
            "query": query_text,
            "precision": prec,
            "latency_ms": latency_ms,
            "top_categories": top_cats,
            "status": status,
        })

        print(f"  [{i+1:2d}] {status:3s} p@{args.k}={prec:.2f}  {latency_ms:6.1f}ms  {query_text[:50]}")
        print(f"       top cats: {top_cats}")

    print("=" * 80)

    if not precisions:
        print("No successful queries.")
        sys.exit(1)

    sorted_latencies = sorted(latencies)
    p50_idx = len(sorted_latencies) // 2
    p95_idx = int(len(sorted_latencies) * 0.95)

    print(f"\nResults ({len(precisions)}/{len(queries)} queries successful):")
    print(f"  Precision@{args.k}:  mean={statistics.mean(precisions):.3f}  "
          f"min={min(precisions):.3f}  max={max(precisions):.3f}")
    print(f"  Latency (ms):   mean={statistics.mean(latencies):.1f}  "
          f"p50={sorted_latencies[p50_idx]:.1f}  "
          f"p95={sorted_latencies[min(p95_idx, len(sorted_latencies)-1)]:.1f}")

    low_count = sum(1 for d in details if d["status"] == "LOW")
    if low_count:
        print(f"\n  WARNING: {low_count} queries with precision < 0.5")


if __name__ == "__main__":
    main()
