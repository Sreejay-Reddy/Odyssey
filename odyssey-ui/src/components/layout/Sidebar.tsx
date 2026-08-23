const navigation = [
  { icon: "⌂", label: "Dashboard" },
  { icon: "⚿", label: "Keys" },
  { icon: "◎", label: "Targets" },
  { icon: "▷", label: "Executions" },
  { icon: "⚙", label: "Settings" },
  { icon: "ⓘ", label: "About" },
];

type SidebarProps = {
  activePage: string;
  onNavigate: (page: string) => void;
};

export function Sidebar({
  activePage,
  onNavigate,
}: SidebarProps) {
  return (
    <aside className="sidebar odyssey-panel">
      <div className="sidebar-brand">
          <img
            src="/dotted_logo_lettering.svg"
            alt="Odyssey"
            className="sidebar-logo"
          />
      </div>

      <nav className="sidebar-nav">
        {navigation.map((item) => (
          <button
            key={item.label}
            className={`nav-item ${
              activePage === item.label ? "active" : ""
            }`}
            onClick={() => onNavigate(item.label)}
          >
            <span className="nav-icon">{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </nav>
    </aside>
  );
}