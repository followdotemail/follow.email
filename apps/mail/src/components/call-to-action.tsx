import { Button } from "@/components/ui/button";
import Link from "next/link";

export default function CallToAction() {
  return (
    <section className=" w-full relative mb-44 mt-14">
       <div className="mx-auto max-w-6xl py-20 px-2 bg-[radial-gradient(ellipse_70%_100%_at_50%_0%,rgba(255,255,255,0.8)_0,rgba(251,146,60,0.6)_40%,transparent_90%)] rounded-3xl">
        <div className="text-center">
          <h2 className=" text-4xl font-semibold lg:text-5xl">
            Start Experience the Future <br/> of Email Today
          </h2>
          <p className="mt-4">Libero sapiente aliquam quibusdam aspernatur.</p>

          <div className="mt-12 flex flex-wrap justify-center gap-4">
            <Button asChild size="lg" variant={"orange"}>
              <Link href="/">
                <span>Get Started</span>
              </Link>
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
