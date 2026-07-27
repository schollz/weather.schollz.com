"use client";

import { Moon, Sun } from "lucide-react";
import { useEffect, useSyncExternalStore } from "react";

const THEME_STORAGE_KEY = "wx-theme";
const THEME_CHANGE_EVENT = "wx-theme-change";

type Theme = "light" | "dark";

function getStoredTheme(): Theme {
  if (typeof window === "undefined") return "dark";
  return window.localStorage.getItem(THEME_STORAGE_KEY) === "light"
    ? "light"
    : "dark";
}

function subscribeToTheme(onStoreChange: () => void) {
  const handleStorage = (event: StorageEvent) => {
    if (event.key === THEME_STORAGE_KEY) onStoreChange();
  };

  window.addEventListener("storage", handleStorage);
  window.addEventListener(THEME_CHANGE_EVENT, onStoreChange);

  return () => {
    window.removeEventListener("storage", handleStorage);
    window.removeEventListener(THEME_CHANGE_EVENT, onStoreChange);
  };
}

function saveTheme(theme: Theme) {
  window.localStorage.setItem(THEME_STORAGE_KEY, theme);
  document.documentElement.dataset.theme = theme;
  window.dispatchEvent(new Event(THEME_CHANGE_EVENT));
}

export default function ThemeToggle({
  showTooltip = false,
}: {
  showTooltip?: boolean;
}) {
  const theme = useSyncExternalStore(
    subscribeToTheme,
    getStoredTheme,
    () => "dark",
  );
  const label = `Switch to ${theme === "dark" ? "light" : "dark"} mode`;

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  return (
    <button
      aria-label={label}
      className={
        showTooltip
          ? "icon-button hover-tip header-tip"
          : "icon-button"
      }
      data-tooltip={showTooltip ? label : undefined}
      onClick={() => saveTheme(theme === "dark" ? "light" : "dark")}
      type="button"
    >
      {theme === "dark" ? (
        <Sun aria-hidden="true" size={14} />
      ) : (
        <Moon aria-hidden="true" size={14} />
      )}
    </button>
  );
}
