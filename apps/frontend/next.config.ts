import type { NextConfig } from "next";

const externalUrl = 'https://api.follow.email'

const nextConfig: NextConfig = {
  /* config options here */
  async rewrites() {
    return [
      {
        source: '/ext-api/:path*',
        destination: `${externalUrl}/:path*`,
      },
    ];
  },
};

export default nextConfig;
