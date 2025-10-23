import { cn } from "@/lib/utils";
import { BarChart3, UserSearch, Lightbulb, Magnet } from "lucide-react";
import type { ReactNode } from "react";
import FeaturesSection from "../feature-section";

const data: { icon: ReactNode; title: string; description: string }[] = [
  {
    icon: <BarChart3 className="size-5" />,
    title: "Freelancers",
    description: "Book more meetings. Close more deals.",
  },
  {
    icon: <UserSearch className="size-5" />,
    title: "Recruiters",
    description: "Reach top talent with personalized outreach.",
  },
  {
    icon: <Lightbulb className="size-5" />,
    title: "Startups",
    description: "Scale outreach without scaling headcount.",
  },
  {
    icon: <Magnet className="size-5" />,
    title: "Agencies",
    description: "Manage clients’ email campaigns in one platform.",
  },
];

export default function UseCase() {
  return (
    <section className="bg-transparent py-20">
      <div className="mx-auto max-w-6xl px-2">
        <div className="mx-auto max-w-3xl text-center">
          <h2 className="text-balance text-4xl font-medium tracking-tight md:text-[42px]">
            Built for growth‑focused teams
          </h2>
          <p className="text-muted-foreground mt-4 text-base md:text-lg">
            Designed for teams that rely on outbound email to drive growth.
          </p>
        </div>

        <FeaturesSection />
      </div>
    </section>
  );
}

function Card({
  icon,
  title,
  description,
  index,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  index: number;
}) {
  return (
    <div
      className={cn(
        "bg-transparent relative p-4 transition-colors duration-200 md:p-8 border",
        // index === 0 && "border-b border-r",
        // index === 1 && "border-b",
        // index === 2 && "border-r",
      )}
    >
      <div className="shadow-sm mb-4 inline-flex h-fit w-fit rounded-(--radius) p-1 bg-transparent">
        <div className="rounded-(--radius) bg-muted/50 flex aspect-square items-center justify-center border p-4">
          {icon}
        </div>
      </div>
      <h3 className="text-xl font-medium tracking-tight">{title}</h3>
      <p className="mt-1 max-w-prose text-base text-muted-foreground">{description}</p>
    </div>
  );
}

<div className="rounded-(--radius) bg-muted/50 flex aspect-square items-center justify-center border p-4"></div>
