import Link from "next/link"; // ★追加
import OccurrenceForm from "@/components/OccurrenceForm";

export default function Home() {
  return (
    <main className="min-h-screen bg-gray-50 py-12 px-4">
      <div className="max-w-3xl mx-auto">
        <div className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold text-gray-800">
            生物多様性DB オカレンス登録
          </h1>
          {/* ★一覧へのリンクボタンを追加 */}
          <Link href="/occurrences" className="px-4 py-2 bg-white border border-gray-300 rounded text-blue-600 hover:bg-blue-50 font-bold">
            一覧を見る →
          </Link>
        </div>
        
        <OccurrenceForm />
      </div>
    </main>
  );
}
