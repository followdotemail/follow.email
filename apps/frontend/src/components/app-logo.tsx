import Image from "next/image";
import React from "react";

const AppLogo = () => {
  return (
    <div className="relative flex h-8 w-8 items-center justify-center overflow-hidden rounded-lg bg-transparent">
      {/* <Link size={16} /> */}
      <Image
        src="/logo.svg"
        alt="follow.email"
        fill
        className="object-contain p-[2px]"
        priority
      />
    </div>
  );
};

export default AppLogo;
