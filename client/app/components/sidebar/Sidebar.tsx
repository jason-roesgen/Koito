import { ExternalLink, History, Home, Info, Repeat, Trophy } from "lucide-react";
import SidebarSearch from "./SidebarSearch";
import SidebarItem from "./SidebarItem";
import SidebarSettings from "./SidebarSettings";

export default function Sidebar() {
  const iconSize = 18;

  return (
    <div
      className="
            z-50
            flex
            sm:flex-col
            justify-between
            sm:fixed
            sm:top-0
            sm:left-0
            sm:h-screen
            h-auto
            sm:w-auto
            w-full
            border-b
            sm:border-b-0
            sm:border-r
            border-(--color-bg-tertiary)
            pt-2
            sm:py-10
            sm:px-3
            px-4
            bg-(--color-bg)
            text-(--color-fg-secondary)
        "
    >
      <div className="flex gap-2 sm:flex-col sm:gap-4">
        <SidebarSearch size={iconSize} />
        <div className="hidden sm:flex"></div>
        <SidebarItem
          space={10}
          to="/"
          name="Home"
          onClick={() => {}}
          modal={<></>}
        >
          <Home size={iconSize} />
        </SidebarItem>
        <SidebarItem
          space={10}
          to="/rewind"
          name="Rewind"
          onClick={() => {}}
          modal={<></>}
        >
          <History size={iconSize} />
        </SidebarItem>
        <SidebarItem space={10} to="/hall-of-fame" name="Hall of Fame" onClick={() => {}} modal={<></>}>
          <Trophy size={iconSize} aria-hidden="true" />
          <span className="sr-only">Hall of Fame</span>
        </SidebarItem>
        <SidebarItem space={10} to="/listening-evolution" name="Listening Evolution" onClick={() => {}} modal={<></>}>
          <Repeat size={iconSize} aria-hidden="true" />
          <span className="sr-only">Listening Evolution</span>
        </SidebarItem>
      </div>
      <div className="flex gap-4 sm:flex-col">
        <SidebarItem
          icon
          keyHint={<ExternalLink size={14} />}
          space={22}
          externalLink
          to="https://koito.io"
          name="About"
          onClick={() => {}}
          modal={<></>}
        >
          <Info size={iconSize} className="relative" />
        </SidebarItem>
        <SidebarSettings size={iconSize} />
      </div>
    </div>
  );
}
