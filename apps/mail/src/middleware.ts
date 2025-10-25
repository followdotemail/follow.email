import { clerkMiddleware, createRouteMatcher } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";

// Define public routes
const isPublicRoute = createRouteMatcher([
  "/sign-in(.*)",
  "/sign-up(.*)",
  "/api/webhook(.*)",
  "/",
]);

export default clerkMiddleware(async (auth, request) => {
  const { userId } = await auth();
  
  if (isPublicRoute(request)) {
    console.log(`Public route accessed: ${request.url}`);
    
    // If user is authenticated and trying to access public routes, redirect to /mail/inbox
    // if (userId) {
    //   return NextResponse.redirect(new URL("/mail/inbox", request.url));
    // }
    
    // Allow access without protection for auth routes
    return;
  } else {
    console.log(`Protected route accessed: ${request.url}`);
    // For protected routes, check if user is authenticated
    // Since we're using separate backend, we'll rely on backend to validate tokens
    await auth.protect();
  }
});

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
