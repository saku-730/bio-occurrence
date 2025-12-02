"use client";

import Link from "next/link";
import { useAuth } from "@/contexts/AuthContext";
import { LogOut, User, Database } from "lucide-react";

export default function Header() {
  const { user, logout } = useAuth();

  return (
    <header className="bg-white border-b border-gray-200 sticky top-0 z-50 shadow-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16 items-center">
          {/* 左側: ロゴ（トップページへのリンク） */}
          <Link 
            href="/" 
            className="text-xl font-black text-blue-600 flex items-center gap-2 hover:opacity-80 transition-opacity"
          >
            <Database className="h-6 w-6" />
            <span>Bio Occurrence DB</span>
          </Link>

          {/* 右側: ユーザー情報とログアウト */}
          <div className="flex items-center gap-4">
            {user ? (
              <>
                <div className="flex items-center gap-2 text-sm text-gray-600 bg-gray-50 px-3 py-1.5 rounded-full">
                  <User className="h-4 w-4 text-gray-400" />
                  <span className="font-medium">{user.username}</span>
                </div>
                
                <button
                  onClick={logout}
                  className="flex items-center gap-1.5 px-3 py-2 text-sm font-bold text-red-600 hover:bg-red-50 rounded-md transition-colors"
                  title="ログアウト"
                >
                  <LogOut className="h-4 w-4" />
                  <span className="hidden sm:inline">ログアウト</span>
                </button>
              </>
            ) : (
              <div className="flex gap-4 text-sm font-medium">
                <Link href="/login" className="text-gray-600 hover:text-blue-600 transition-colors">
                  ログイン
                </Link>
                <Link 
                  href="/register" 
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                >
                  新規登録
                </Link>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
