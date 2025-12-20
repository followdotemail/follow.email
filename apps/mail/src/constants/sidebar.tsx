import { InboxIcon } from "@/utils/icons/inbox";
import { DraftIcon } from "@/utils/icons/draft";
import { SentIcon } from "@/utils/icons/sent";
import { BinIcon } from "@/utils/icons/bin";
import { ArchiveIcon } from "@/utils/icons/archive";
import { SpamIcon } from "@/utils/icons/spam";
import { FeedbackIcon } from "@/utils/icons/feedback";
import { SettingIcon } from "@/utils/icons/setting";
import { ClockIcon } from "@/utils/icons/clock";


export const navItems = {
    main: [
      {
        title: "Inbox",
        label: "100",
        icon: InboxIcon,
        variant: "default" as const,
      },
      {
        title: "Draft",
        label: "2",
        icon: DraftIcon,
        variant: "ghost" as const,
      },
      {
        title: "Sent",
        label: "",
        icon: SentIcon,
        variant: "ghost" as const,
      },
      {
        title: "Schedule",
        label: "7",
        icon: ClockIcon,
        variant: "ghost" as const,
      },
      {
        title: "Trash",
        label: "5",
        icon: BinIcon,
        variant: "ghost" as const,
      },
      {
        title: "Archive",
        label: "",
        icon: ArchiveIcon,
        variant: "ghost" as const,
      },
      {
        title: "Spam",
        label: "24",
        icon: SpamIcon,
        variant: "ghost" as const,
      },
    ],
    secondary: [
      {
        title: "Feedback",
        label: "",
        icon: FeedbackIcon,
        variant: "ghost" as const,
      },
      {
        title: "Settings",
        label: "",
        icon: SettingIcon,
        variant: "ghost" as const,
      },
    ],
  };
