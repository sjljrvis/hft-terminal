import { Sun, Moon , WaveSine } from "phosphor-react";

function Header({ theme, onThemeChange }) {
  const handleThemeClick = (nextTheme) => {
    onThemeChange(nextTheme);
  };

  return (
    <header className="app-header">
      <div className="app-header__title">
        <WaveSine size={14} weight="regular" />
        HFT Dashboard
      </div>
      <div className="theme-toggle" aria-label="Theme selection">
        <button
          type="button"
          data-theme="light"
          className={theme === "light" ? "is-active" : ""}
          onClick={() => handleThemeClick("light")}
          aria-pressed={theme === "light"}
        >
          <Sun size={14} weight="regular" />
          <span className="theme-toggle__label">Light</span>
        </button>
        <button
          type="button"
          data-theme="dark"
          className={theme === "dark" ? "is-active" : ""}
          onClick={() => handleThemeClick("dark")}
          aria-pressed={theme === "dark"}
        >
          <Moon size={14} weight="regular" />
          <span className="theme-toggle__label">Dark</span>
        </button>
      </div>
    </header>
  );
}

export default Header;
