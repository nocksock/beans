const { execFileSync } = require("node:child_process");
const path = require("node:path");

// Documentation in this folder is plain Markdown. Do not run the Liquid
// templating engine over it: code samples contain Go template/printf syntax
// such as `{%q %q}` and `{{Name: "..."}}` that Liquid would try to parse.
module.exports = function (eleventyConfig) {
  eleventyConfig.addPassthroughCopy("assets");
  eleventyConfig.addGlobalData("layout", "layout.njk");

  eleventyConfig.addGlobalData("buildInfo", () => {
    let branch = "unknown";
    try {
      branch = execFileSync("git", ["rev-parse", "--abbrev-ref", "HEAD"], {
        cwd: __dirname,
        encoding: "utf8",
      }).trim();
    } catch {
      // not a git checkout; leave the fallback
    }
    return { cwd: process.cwd(), branch };
  });

  // Group pages by their directory (relative to this folder), root first,
  // then alphabetically; pages within a group are newest-first.
  eleventyConfig.addCollection("nav", (collectionApi) => {
    const groups = new Map();

    for (const item of collectionApi.getAll()) {
      if (item.page.url === "/") continue; // skip the home page

      const dir =
        path.posix.dirname(item.page.filePathStem).replace(/^\/+/, "") || ".";

      if (!groups.has(dir)) groups.set(dir, []);
      groups.get(dir).push({
        title: item.data.title || humanize(item.page.fileSlug),
        url: item.page.url,
      });
    }

    return [...groups.entries()]
      .sort(([a], [b]) => (a === "." ? -1 : b === "." ? 1 : a.localeCompare(b)))
      .map(([name, items]) => ({
        name,
        items: items
          .sort((a, b) => b.url.localeCompare(a.url))
          .map(({ title, url }) => ({ title, url })),
      }));
  });
  return {
    markdownTemplateEngine: false,
  };
};

function humanize(slug) {
  return slug
    .replace(/^\d{4}-\d{2}-\d{2}-/, "")
    .replace(/-/g, " ")
    .toLowerCase()
    .replace(/^./, (c) => c.toUpperCase());
}
