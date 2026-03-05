"use client";

import { useRouter } from "next/navigation";
import { useLayoutEffect } from "react";

export default function HomePage() {
  const router = useRouter();

  useLayoutEffect(() => {
    router.replace("/new");
  }, [router]);

  return null;
}
