// Material Symbols (Rounded, weight 400), inlined so only the icons in use ship.
// Source: https://fonts.google.com/icons (Apache License 2.0).
import styles from "./Icon.module.css";

const PATHS = {
  brightness_6:
    "M346.16-160H220q-24.75 0-42.37-17.63Q160-195.25 160-220v-125.59L68-438q-9-9-13-19.81-4-10.82-4-22Q51-491 55-502q4-11 13-20l92-92.41V-740q0-24.75 17.63-42.38Q195.25-800 220-800h125.59L438-892q9-9 20.5-13t22.7-4q11.19 0 22.02 4.7 10.82 4.69 19.78 13.3l91 91h126q24.75 0 42.38 17.62Q800-764.75 800-740v125.59L892-522q9 9 13 19.81 4 10.82 4 22 0 11.19-4 22.19-4 11-13 20l-92 92.41V-220q0 24.75-17.62 42.37Q764.75-160 740-160H614l-91 90q-8.96 8.13-19.78 12.57Q492.39-53 481.2-53q-11.2 0-22.16-4.43Q448.07-61.87 439-70l-92.84-90ZM370-220l111 107 107.92-107H740v-151l109-109-109-109v-151H589L481-849 371-740H220v151L111-480l109 109v151h150Zm111-66q81 0 138-57.05 57-57.06 57-138Q676-562 618.96-619 561.92-676 481-676v390Z",
  check:
    "m378-332 363-363q9-9 21.5-9t21.5 9q9 9 9 21.5t-9 21.5L399-267q-9 9-21 9t-21-9L175-449q-9-9-8.5-21.5T176-492q9-9 21.5-9t21.5 9l159 160Z",
  expand_more:
    "M480-357q-6 0-11-2t-10-7L261-564q-9-9-9-21t9-21q9-9 21.5-9t21.5 9l176 176 176-176q9-9 21-9t21 9q9 9 9 21.5t-9 21.5L501-366q-5 5-10 7t-11 2Z",
  open_in_new:
    "M180-120q-24 0-42-18t-18-42v-600q0-24 18-42t42-18h249q12.75 0 21.38 8.68 8.62 8.67 8.62 21.5 0 12.82-8.62 21.32-8.63 8.5-21.38 8.5H180v600h600v-249q0-12.75 8.68-21.38 8.67-8.62 21.5-8.62 12.82 0 21.32 8.62 8.5 8.63 8.5 21.38v249q0 24-18 42t-42 18H180Zm600-617L403-360q-9 9-21 8.5t-21-9.5q-9-9-9-21t9-21l377-377H549q-12.75 0-21.37-8.68-8.63-8.67-8.63-21.5 0-12.82 8.63-21.32 8.62-8.5 21.37-8.5h261q12.75 0 21.38 8.62Q840-822.75 840-810v261q0 12.75-8.68 21.37-8.67 8.63-21.5 8.63-12.82 0-21.32-8.63-8.5-8.62-8.5-21.37v-188Z",
} as const;

export type IconName = keyof typeof PATHS;

type Props = {
  name: IconName;
  /** 24 is the M3 default; 18 is used inside buttons and chips. */
  size?: 18 | 24;
  className?: string;
};

/** Decorative by default: pair it with visible text or an aria-label on the parent. */
export function Icon({ name, size = 24, className }: Props) {
  return (
    <svg
      className={[styles.icon, className].filter(Boolean).join(" ")}
      width={size}
      height={size}
      viewBox="0 -960 960 960"
      aria-hidden="true"
      focusable="false"
    >
      <path d={PATHS[name]} />
    </svg>
  );
}
