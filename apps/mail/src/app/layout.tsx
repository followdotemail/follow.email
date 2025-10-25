import type { Metadata } from "next";
import { Geist, Geist_Mono, Roboto } from "next/font/google";
import "@/app/globals.css";
import { dark } from "@clerk/themes";
import { ThemeProvider } from "@/components/theme-provider";
import { ClerkProvider } from "@clerk/nextjs";
import { Provider } from "./provider";

const geist = Geist({
  variable: "--font-geist",
  subsets: ["latin"],
  display: "swap",
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "Follow.Email - Modern Email Client",
    template: "%s | Follow.Email",
  },
  description:
    "A modern, fast, and intuitive email client for managing your emails efficiently. Stay organized with powerful features and a beautiful interface.",
  keywords: [
    "email",
    "email client",
    "inbox",
    "mail",
    "webmail",
    "email management",
    "productivity",
    "follow email",
  ],
  authors: [{ name: "Follow.Email" }],
  creator: "Follow.Email",
  publisher: "Follow.Email",
  metadataBase: new URL("https://follow.email"),
  openGraph: {
    type: "website",
    locale: "en_US",
    url: "/",
    title: "Follow.Email - Modern Email Client",
    description:
      "A modern, fast, and intuitive email client for managing your emails efficiently.",
    siteName: "Follow.Email",
    images: [
      {
        url: "/images/email-preview.png",
        width: 1200,
        height: 630,
        alt: "Follow.Email Preview",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Follow.Email - Modern Email Client",
    description:
      "A modern, fast, and intuitive email client for managing your emails efficiently.",
    images: ["/images/email-preview.png"],
    creator: "@followemail",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
  icons: {
    icon: "/favicon.ico",
    shortcut: "/favicon.ico",
    apple: "/apple-touch-icon.png",
  },
  manifest: "/site.webmanifest",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <ClerkProvider
      appearance={{
        baseTheme: dark,
      }}
    >
      <html lang="en" suppressHydrationWarning>
        <body className={`${geist.variable} ${geistMono.variable}`} suppressHydrationWarning>
          <ThemeProvider
            attribute="class"
            defaultTheme="dark"
            enableSystem
            disableTransitionOnChange
          >
            <Provider>{children}</Provider>
          </ThemeProvider>
        </body>
      </html>
    </ClerkProvider>
  );
}
