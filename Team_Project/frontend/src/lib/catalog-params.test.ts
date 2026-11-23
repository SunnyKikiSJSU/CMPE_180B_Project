import { describe, expect, it } from "vitest";

import { catalogHref, hasExternalTerms, parseCatalogParams, toExternalSearch } from "@/lib/catalog-params";

const parse = (qs: string) => parseCatalogParams(new URLSearchParams(qs));

describe("catalog URL state", () => {
  it("defaults to the Open Library tab, page 1, relevance", () => {
    expect(parse("")).toMatchObject({ tab: "openlibrary", q: "", page: 1, sort: "relevance" });
    expect(hasExternalTerms(parse(""))).toBe(false);
  });

  it("rejects bad pages and sorts the tab does not support", () => {
    expect(parse("page=-3&sort=rating")).toMatchObject({ page: 1, sort: "relevance" });
    expect(parse("tab=library&sort=rating").sort).toBe("rating");
    expect(parse("page=500").page).toBe(100);
    expect(parse("tab=library&page=500").page).toBe(500);
  });

  it("keeps only the current tab's filters", () => {
    expect(parse("tab=library&q=night&isbn=123&authorId=a1")).toMatchObject({ q: "night", isbn: "", authorId: "a1" });
    expect(parse("q=dune&categoryId=c1&isbn=978-0-441-01359-3")).toMatchObject({ categoryId: "", isbn: "9780441013593" });
  });

  it("builds short URLs that round-trip", () => {
    expect(catalogHref({ q: "dune" })).toBe("/catalog?q=dune");
    expect(catalogHref({ tab: "library", q: "night owl", sort: "rating", page: 2 })).toBe("/catalog?tab=library&q=night+owl&sort=rating&page=2");
    expect(catalogHref({ tab: "openlibrary", authorId: "a1", page: 1 })).toBe("/catalog");
    const state = parse(catalogHref({ subject: "fantasy", sort: "newest", page: 3 }).split("?")[1]);
    expect(toExternalSearch(state)).toMatchObject({ category: "fantasy", sort: "newest", page: 3, pageSize: 20 });
  });
});
