import type { Config } from "@react-router/dev/config";

export default {
  // Server-rendered so search engines and link previews (LINE, Facebook) get real HTML.
  ssr: true,
} satisfies Config;
