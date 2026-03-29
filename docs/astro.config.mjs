import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

export default defineConfig({
  site: "https://maximerivest.github.io",
  base: "/mcp2cli",
  integrations: [
    starlight({
      title: "mcp2cli",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/MaximeRivest/mcp2cli",
        },
      ],
      sidebar: [
        { label: "Overview", slug: "overview" },
        { label: "Installation", slug: "installation" },
        { label: "Quick Start", slug: "quick-start" },
        {
          label: "Guides",
          items: [
            { label: "Using Tools", slug: "guides/tools" },
            { label: "Remote Servers", slug: "guides/remote" },
            { label: "Background Mode", slug: "guides/background" },
            { label: "Interactive Shell", slug: "guides/shell" },
            { label: "Exposed Commands", slug: "guides/expose" },
            { label: "Output Formats", slug: "guides/output" },
            { label: "Arguments", slug: "guides/arguments" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "All Commands", slug: "reference/commands" },
            { label: "Configuration", slug: "reference/config" },
            { label: "Troubleshooting", slug: "reference/troubleshooting" },
          ],
        },
      ],
    }),
  ],
});
