// M3 menu: https://m3.material.io/components/menus/overview
// Base UI: https://base-ui.com/react/components/menu
import { Menu } from "@base-ui/react/menu";
import { useFetcher } from "react-router";
import { useI18n } from "~/i18n/i18n";
import { isLocale, LOCALES, type Locale } from "~/i18n/locales";
import { isThemeChoice, type ThemeChoice } from "~/lib/preferences";
import { Divider } from "../Divider/Divider";
import { Icon } from "../Icon/Icon";
import { IconButton } from "../IconButton/IconButton";
import { Tooltip } from "../Tooltip/Tooltip";
import styles from "./SettingsMenu.module.css";

const THEMES: ThemeChoice[] = ["system", "light", "dark"];

/** Appearance and language. Both are saved as cookies by the /preferences action. */
export function SettingsMenu({ theme }: { theme: ThemeChoice }) {
  const { m, locale } = useI18n();
  const fetcher = useFetcher();

  // Show the pending choice immediately while the cookie is being saved.
  const pendingTheme = fetcher.formData?.get("theme");
  const pendingLocale = fetcher.formData?.get("locale");
  const themeValue = isThemeChoice(pendingTheme) ? pendingTheme : theme;
  const localeValue = isLocale(pendingLocale) ? pendingLocale : locale;

  function chooseTheme(next: ThemeChoice) {
    if (next === "system") delete document.documentElement.dataset.theme;
    else document.documentElement.dataset.theme = next;
    fetcher.submit({ theme: next }, { method: "post", action: "/preferences" });
  }

  function chooseLocale(next: Locale) {
    fetcher.submit({ locale: next }, { method: "post", action: "/preferences" });
  }

  return (
    <Menu.Root>
      <Tooltip label={m.settings.label}>
        <Menu.Trigger render={<IconButton icon="brightness_6" label={m.settings.label} />} />
      </Tooltip>
      <Menu.Portal>
        <Menu.Positioner align="end" sideOffset={4}>
          <Menu.Popup className={styles.popup}>
            <Menu.Group>
              <Menu.GroupLabel className={`${styles.groupLabel} md-label-medium`}>
                {m.settings.appearance}
              </Menu.GroupLabel>
              <Menu.RadioGroup
                value={themeValue}
                onValueChange={(next) => {
                  if (isThemeChoice(next)) chooseTheme(next);
                }}
              >
                {THEMES.map((value) => (
                  <RadioItem key={value} value={value} label={m.settings[value]} />
                ))}
              </Menu.RadioGroup>
            </Menu.Group>
            <div className={styles.divider}>
              <Divider />
            </div>
            <Menu.Group>
              <Menu.GroupLabel className={`${styles.groupLabel} md-label-medium`}>
                {m.settings.language}
              </Menu.GroupLabel>
              <Menu.RadioGroup
                value={localeValue}
                onValueChange={(next) => {
                  if (isLocale(next)) chooseLocale(next);
                }}
              >
                {(Object.keys(LOCALES) as Locale[]).map((value) => (
                  <RadioItem
                    key={value}
                    value={value}
                    label={LOCALES[value].label}
                    lang={LOCALES[value].htmlLang}
                  />
                ))}
              </Menu.RadioGroup>
            </Menu.Group>
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}

function RadioItem({ value, label, lang }: { value: string; label: string; lang?: string }) {
  return (
    <Menu.RadioItem value={value} className={`${styles.item} md-label-large`} lang={lang}>
      <span className={styles.indicator}>
        <Menu.RadioItemIndicator>
          <Icon name="check" />
        </Menu.RadioItemIndicator>
      </span>
      {label}
    </Menu.RadioItem>
  );
}
