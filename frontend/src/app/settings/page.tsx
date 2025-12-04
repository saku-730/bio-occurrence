"use client";

import { useAuth } from "@/contexts/AuthContext";
import { User, Mail, Shield } from "lucide-react";

export default function SettingsPage() {
  const { user } = useAuth();

  if (!user) return null; // AuthGuardが守ってくれるので基本ここには来ない

  return (
    <main className="min-h-screen bg-gray-50 py-12 px-4">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-3xl font-bold text-gray-800 mb-8">アカウント設定</h1>

        <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
          <div className="p-6 border-b border-gray-100 bg-gray-50/50">
            <h2 className="text-lg font-bold text-gray-700 flex items-center gap-2">
              <User className="h-5 w-5 text-blue-600" />
              プロフィール情報
            </h2>
            <p className="text-sm text-gray-500 mt-1">
              登録されている基本情報
            </p>
          </div>

          <div className="p-6 space-y-6">
            {/* ユーザー名 */}
            <div>
              <label className="block text-sm font-medium text-gray-500 mb-1">
                ユーザー名
              </label>
              <div className="text-lg font-bold text-gray-900 flex items-center gap-2">
                {user.username}
              </div>
            </div>

            {/* メールアドレス */}
            <div>
              <label className="block text-sm font-medium text-gray-500 mb-1">
                メールアドレス
              </label>
              <div className="text-lg text-gray-900 flex items-center gap-2">
                <Mail className="h-4 w-4 text-gray-400" />
                {user.email}
              </div>
            </div>

            {/* ユーザーID (デバッグ用に見たいとき便利) */}
            <div>
              <label className="block text-sm font-medium text-gray-500 mb-1">
                ユーザーID (UUID)
              </label>
              <div className="text-sm font-mono text-gray-400 bg-gray-50 p-2 rounded border border-gray-100 inline-block">
                {user.id}
              </div>
            </div>
          </div>
        </div>
        
        {/* 将来的な機能拡張エリア */}
        <div className="mt-6 p-4 rounded-lg bg-blue-50 text-blue-700 text-sm flex items-start gap-3">
          <Shield className="h-5 w-5 mt-0.5 flex-shrink-0" />
          <div>
            <p className="font-bold">パスワードの変更について</p>
            <p className="opacity-80">
              現在はパスワード変更機能は実装されていない。実装予定
            </p>
          </div>
        </div>

      </div>
    </main>
  );
}
