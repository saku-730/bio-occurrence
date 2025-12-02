"use client";

import { useEffect } from "react";
import { useAuth } from "@/contexts/AuthContext";
import { useRouter, usePathname } from "next/navigation";
import { Loader2 } from "lucide-react";

// ログインしていなくてもアクセスできるページ
const PUBLIC_PATHS = ["/login", "/register"];

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    // 1. ロード中は判定しない
    if (isLoading) return;

    // 2. 今いるページが「公開ページ」かどうかチェック
    const isPublicPath = PUBLIC_PATHS.includes(pathname);

    // 3. ユーザーがいなくて、かつ非公開ページなら、ログイン画面へ飛ばす
    if (!user && !isPublicPath) {
      router.push("/login");
    }

    // (おまけ) 逆に、ログイン済みなのにログイン画面に来たらトップへ飛ばす
    if (user && isPublicPath) {
      router.push("/");
    }

  }, [user, isLoading, pathname, router]);

  // 判定中のローディング表示
  // これがないと、リダイレクトされる前に一瞬保護ページが見えてしまう（チラつき防止）
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  // リダイレクト対象の場合は何も表示しない（useEffectで飛ぶのを待つ）
  if (!user && !PUBLIC_PATHS.includes(pathname)) {
    return null;
  }

  // 問題なければページの中身を表示
  return <>{children}</>;
}
