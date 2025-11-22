import { redirect } from "next/navigation";
import { auth } from "@clerk/nextjs/server";
import { checkUserConsentStatus } from "@/server/api/user-status";

export default async function Mail() {
  const { getToken } = await auth();
  const token = await getToken();

  if (!token) {
    return redirect("/sign-in");
  }

  const { isConnected } = await checkUserConsentStatus(token);

  console.log("isConnected", isConnected);

  return redirect(isConnected === true ? "/mail/inbox" : "/mail/onboarding");
}
