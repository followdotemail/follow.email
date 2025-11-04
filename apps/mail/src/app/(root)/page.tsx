import UseCase from "@/components/app/use-case";
import { BentoSection } from "@/components/bento-section";
import FeaturesSection from "@/components/feature-section";
import Features from "@/components/features-11";
import HeroSection from "@/components/hero-section";
import IntegrationsSection from "@/components/integrations-7";
import PricingSection from "@/components/pricing-section";
import { Testimonials } from "@/components/demo-testimonial";
import FAQsFour from "@/components/faqs-4";
import CallToAction from "@/components/call-to-action";
import FooterSection from "@/components/footer-three";

export default function Home() {
  return (
    <main>
      <HeroSection />
      <BentoSection />
      <UseCase />
      {/* <Features /> */}
      <PricingSection />
      <Testimonials />
      <FAQsFour />
      {/* <IntegrationsSection /> */}
      <CallToAction />
      <FooterSection />
    </main>
  );
}
