"use client";

import { useEffect, useState, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation"; // ★ここが変わった
import { Loader2, ArrowLeft, Tag } from "lucide-react";
import { Pencil, Trash2 } from "lucide-react"

type Trait = {
  id: string;
  label: string;
};

type DetailData = {
  id: string;
  taxon_label: string;
  remarks: string;
  traits: Trait[];
};

// 中身のコンポーネント（データ取得ロジック）
function DetailContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  
  // URLの ?id=... を取得するのだ
  const id = searchParams.get("id");

  const [data, setData] = useState<DetailData | null>(null);
  const [loading, setLoading] = useState(true);
  const handleDelete = async () => {
    if (!confirm("削除してもよろしいですか？この操作は取り消せません。")) return;
    
    try {
      const res = await fetch(`http://localhost:8080/api/occurrences/${id}`, {
        method: "DELETE",
      });
      if (!res.ok) throw new Error("削除失敗");
      
      alert("削除したのだ！");
      router.push("/occurrences"); // 一覧に戻る
    } catch (err) {
      alert("エラーなのだ");
    }
  };

  useEffect(() => {
    if (!id) return;

    const fetchDetail = async () => {
      try {
        const res = await fetch(`http://localhost:8080/api/occurrences/${id}`);
        if (!res.ok) throw new Error("取得失敗");
        const json = await res.json();
        setData(json);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchDetail();
  }, [id]);

  if (!id) return <div className="p-10 text-center">IDが指定されていません</div>;
  if (loading) return <div className="p-10 flex justify-center"><Loader2 className="animate-spin" /></div>;
  if (!data) return <div className="p-10 text-center">データが見つからないのだ...</div>;

  return (
    <div className="max-w-3xl mx-auto bg-white p-8 rounded-xl shadow-lg border border-gray-100">
      <div className="flex justify-between items-start mb-6">
        <button onClick={() => router.back()} className="flex items-center text-gray-500 hover:text-blue-600 transition-colors">
          <ArrowLeft className="h-4 w-4 mr-1" /> 一覧に戻る
        </button>

        {/* 操作ボタン群 */}
        <div className="flex gap-2">
          <button 
            onClick={() => router.push(`/occurrences/edit?id=${id}`)}
            className="flex items-center px-3 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-md text-sm font-bold transition-colors"
          >
            <Pencil className="h-4 w-4 mr-1" /> 編集
          </button>
          <button 
            onClick={handleDelete}
            className="flex items-center px-3 py-2 bg-red-50 hover:bg-red-100 text-red-600 rounded-md text-sm font-bold transition-colors"
          >
            <Trash2 className="h-4 w-4 mr-1" /> 削除
          </button>
        </div>
      </div>

      <h1 className="text-4xl font-bold text-gray-900 mb-2">
        {data.taxon_label}
      </h1>
      <div className="text-xs text-gray-400 font-mono mb-8 break-all">
        URI: {data.id}
      </div>

      <div className="mb-8">
        <h2 className="text-sm font-bold text-gray-500 uppercase tracking-wider mb-3">特徴・形質</h2>
        <div className="flex flex-wrap gap-2">
          {data.traits.length > 0 ? (
            data.traits.map((t) => (
              <span key={t.id} className="inline-flex items-center px-3 py-1 bg-green-100 text-green-800 rounded-full text-sm">
                <Tag className="h-3 w-3 mr-1" />
                {t.label}
              </span>
            ))
          ) : (
            <span className="text-gray-400">登録された特徴はありません</span>
          )}
        </div>
      </div>

      <div className="bg-gray-50 p-6 rounded-lg border border-gray-100">
        <h2 className="text-sm font-bold text-gray-500 uppercase tracking-wider mb-2">メモ</h2>
        <p className="text-gray-700 whitespace-pre-wrap leading-relaxed">
          {data.remarks || "（記述なし）"}
        </p>
      </div>
    </div>
  );
}

// メインコンポーネント（Suspenseで囲むのが必須ルール！）
export default function OccurrenceDetail() {
  return (
    <main className="min-h-screen bg-gray-50 py-10 px-4">
      <Suspense fallback={<div className="text-center p-10">読み込み中...</div>}>
        <DetailContent />
      </Suspense>
    </main>
  );
}
