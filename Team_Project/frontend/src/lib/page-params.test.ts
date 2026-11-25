import { describe, expect, it } from "vitest";

import { hrefWith, namesHref, pageParam, parseNamesParams, parseReportParams, parseReviewsParams, reportHref, reviewsHref } from "@/lib/page-params";

const reader = (query: string) => new URLSearchParams(query);

describe("pageParam", () => {
  it("accepts whole pages 1-10000 only", () => {
    expect(pageParam("3")).toBe(3);
    for (const bad of [null, "", "0", "-2", "1.5", "abc", "10001"]) expect(pageParam(bad)).toBe(1);
  });
});

describe("hrefWith", () => {
  it("leaves out empty and default values", () => {
    expect(hrefWith("/x", { a: "1", b: "", c: undefined, d: 2 }, { d: 2 })).toBe("/x?a=1");
    expect(hrefWith("/x", {})).toBe("/x");
  });
});

describe("reviews params", () => {
  it("round-trips and falls back to defaults", () => {
    expect(parseReviewsParams(reader("tab=ratings&page=2"))).toEqual({ tab: "ratings", page: 2 });
    expect(parseReviewsParams(reader("tab=nope&page=x"))).toEqual({ tab: "reviews", page: 1 });
    expect(reviewsHref({ tab: "reviews", page: 1 })).toBe("/reviews");
    expect(reviewsHref({ tab: "ratings", page: 3 })).toBe("/reviews?tab=ratings&page=3");
  });
});

describe("authors params", () => {
  it("trims the search and keeps only known tabs", () => {
    expect(parseNamesParams(reader("tab=categories&q=%20sci%20&page=2"))).toEqual({ tab: "categories", q: "sci", page: 2 });
    expect(parseNamesParams(reader("tab=books"))).toEqual({ tab: "authors", q: "", page: 1 });
    expect(namesHref({ tab: "authors", q: "", page: 1 })).toBe("/authors");
    expect(namesHref({ tab: "categories", q: "art", page: 2 })).toBe("/authors?tab=categories&q=art&page=2");
  });
});

describe("report params", () => {
  it("uses the API defaults per tab", () => {
    expect(parseReportParams(reader(""))).toEqual({ tab: "mine", days: 30, minRatings: 5, sql: "" });
    expect(parseReportParams(reader("tab=top-rated"))).toMatchObject({ tab: "top-rated", minRatings: 1 });
    expect(parseReportParams(reader("tab=authors&minRatings=0&days=7"))).toEqual({ tab: "authors", days: 7, minRatings: 0, sql: "" });
  });

  it("refuses values outside the offered options", () => {
    expect(parseReportParams(reader("tab=popular&days=400&minRatings=-1"))).toEqual({ tab: "popular", days: 30, minRatings: 5, sql: "" });
    expect(parseReportParams(reader("tab=queries&sql=q05")).sql).toBe("q05");
    expect(parseReportParams(reader("tab=queries&sql=<script>")).sql).toBe("");
    expect(parseReportParams(reader("tab=mine&sql=q05")).sql).toBe("");
  });

  it("keeps only the parameters the tab uses", () => {
    expect(reportHref({ tab: "mine" })).toBe("/reports");
    expect(reportHref({ tab: "popular", days: 30, minRatings: 3 })).toBe("/reports?tab=popular");
    expect(reportHref({ tab: "popular", days: 90 })).toBe("/reports?tab=popular&days=90");
    expect(reportHref({ tab: "top-rated", minRatings: 1, days: 90 })).toBe("/reports?tab=top-rated");
    expect(reportHref({ tab: "categories", minRatings: 0 })).toBe("/reports?tab=categories&minRatings=0");
    expect(reportHref({ tab: "queries", sql: "q05" })).toBe("/reports?tab=queries&sql=q05");
    expect(reportHref({ tab: "popular", sql: "q05" })).toBe("/reports?tab=popular");
  });
});
