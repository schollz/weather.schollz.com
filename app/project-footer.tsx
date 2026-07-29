const TOOLS = [
  {
    description: "fast, simple, secure file transfer",
    href: "https://getcroc.com",
    name: "croc",
  },
  {
    description: "write together, without the setup",
    href: "https://cowyo.com",
    name: "cowyo",
  },
  {
    description: "yes/no alerts when websites change",
    href: "https://yesnotice.com",
    name: "yesnotice",
  },
];

export default function ProjectFooter() {
  return (
    <footer className="project-footer">
      <div className="project-footer-links">
        <span>
          made by{" "}
          <a
            href="https://github.com/sponsors/schollz"
            rel="noreferrer"
            target="_blank"
          >
            schollz
          </a>
        </span>
        <span aria-hidden="true">·</span>
        <a
          href="https://github.com/schollz/wthrtxt"
          rel="noreferrer"
          target="_blank"
        >
          github
        </a>
      </div>

      <details className="tools-menu">
        <summary>other tools</summary>
        <ul>
          {TOOLS.map((tool) => (
            <li key={tool.href}>
              <a href={tool.href} rel="noreferrer" target="_blank">
                <strong>{tool.name}</strong>
                <span>{tool.description}</span>
              </a>
            </li>
          ))}
        </ul>
      </details>
    </footer>
  );
}
