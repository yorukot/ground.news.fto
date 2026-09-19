import "@fontsource/roboto/latin-400.css";
import "@fontsource/roboto/latin-500.css";
import "@fontsource/roboto/latin-700.css";
import "./theme/colors.css";
import "./theme/tokens.css";
import "./theme/typography.css";
import "./theme/base.css";

import type { ReactNode } from "react";
import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  useNavigation,
  useRouteLoaderData,
} from "react-router";
import type { Route } from "./+types/root";
import { ButtonLink } from "./components/Button/Button";
import { LinearProgress } from "./components/LinearProgress/LinearProgress";
import { SettingsMenu } from "./components/SettingsMenu/SettingsMenu";
import { TooltipProvider } from "./components/Tooltip/Tooltip";
import { TopAppBar } from "./components/TopAppBar/TopAppBar";
import { I18nProvider, useI18n } from "./i18n/i18n";
import { DEFAULT_LOCALE, LOCALES } from "./i18n/locales";
import { readPreferences } from "./lib/preferences";
import styles from "./root.module.css";

export function loader({ request }: Route.LoaderArgs) {
  return readPreferences(request.headers.get("Cookie"));
}

export function Layout({ children }: { children: ReactNode }) {
  // Also used by the error boundary, where loader data may be missing.
  const prefs = useRouteLoaderData<typeof loader>("root");
  const locale = prefs?.locale ?? DEFAULT_LOCALE;
  const theme = prefs?.theme ?? "system";

  return (
    <html lang={LOCALES[locale].htmlLang} data-theme={theme === "system" ? undefined : theme}>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body>
        <I18nProvider locale={locale}>
          <TooltipProvider>
            <div className="root">
              <Shell theme={theme}>{children}</Shell>
            </div>
          </TooltipProvider>
        </I18nProvider>
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

function Shell({ theme, children }: { theme: "system" | "light" | "dark"; children: ReactNode }) {
  const { m } = useI18n();
  const navigation = useNavigation();
  const navigating = navigation.state === "loading" && navigation.location !== undefined;

  return (
    <>
      <a href="#main" className={`${styles.skip} md-label-large`}>
        {m.nav.skipToContent}
      </a>
      <TopAppBar actions={<SettingsMenu theme={theme} />} />
      <div className={styles.progress}>
        {navigating && <LinearProgress label={m.common.loading} />}
      </div>
      <main id="main" className={styles.main}>
        {children}
      </main>
      <footer className={`${styles.footer} md-body-small`}>
        <p>{m.site.tagline}</p>
      </footer>
    </>
  );
}

export default function App() {
  return <Outlet />;
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const { m } = useI18n();
  let title = m.error.genericTitle;
  let body = m.error.genericBody;

  if (isRouteErrorResponse(error)) {
    if (error.status === 404) {
      title = m.error.notFoundTitle;
      body = m.error.notFoundBody;
    } else if (error.status === 502 || error.status === 503) {
      title = m.error.unavailableTitle;
      body = m.error.unavailableBody;
    }
  }

  return (
    <div className={styles.error}>
      <h1 className="md-headline-medium">{title}</h1>
      <p className="md-body-large">{body}</p>
      {import.meta.env.DEV && error instanceof Error && (
        <pre className={`${styles.stack} md-body-small`}>{error.stack}</pre>
      )}
      <div>
        <ButtonLink to="/" variant="tonal">
          {m.error.backHome}
        </ButtonLink>
      </div>
    </div>
  );
}
