"use client";

import { useEffect, useState, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { Loader2, ArrowLeft, ArrowRight, Tag, Pencil, Trash2 } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext"; // ★追加: AuthContextを使う

type Trait = {
  predicate_id: string;
  predicate_label: string;
  value_id: string;
  value_label: string;
};

type DetailData = {
  id: string;
  taxon_label: string;
  remarks: string;
  traits: Trait[];
  owner_name?: string;
  created_at?: string;
};


function DetailContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const { token } = useAuth(); // ★追加: トークンを取り出す
  
  const id = searchParams.get("id");

  const [data, setData] = useState<DetailData | null>(null);
  const [loading, setLoading] = useState(true);

  // 削除処理の修正
  const handleDelete = async () => {
    if (!token) {
      alert("削除するにはログインが必要なのだ！");
      router.push("/login");
      return;
    }

    if (!confirm("本当に削除してもよろしいですか？この操作は取り消せません。")) return;
    
    try {
      const res = await fetch(`http://localhost:8080/api/occurrences/${id}`, {
        method: "DELETE",
        headers: { 
            "Authorization": `Bearer ${token}` // ★追加: これがないと401になる！
        },
      });
      
      if (!res.ok) {
        if (res.status === 401) throw new Error("認証エラー：ログインし直してください");
        throw new Error("削除失敗");
      }
      
      alert("削除したのだ！");
      router.push("/occurrences"); 
    } catch (err: any) {
      alert(err.message || "エラーなのだ");
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
  if (!data) return <div className="p-10 text-center">データが見つからない</div>;

  return (
    <div className="max-w-3xl mx-auto bg-white p-8 rounded-xl shadow-lg border border-gray-100">
      <div className="flex justify-between items-start mb-6">
        <button onClick={() => router.back()} className="flex items-center text-gray-500 hover:text-blue-600 transition-colors">
          <ArrowLeft className="h-4 w-4 mr-1" /> 一覧に戻る
        </button>

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
      
      <div className="text-sm text-gray-600 mb-1">
        登録者: <span className="font-bold">{data.owner_name || "不明"}</span>
      {data.created_at && (
	<div>
	 登録日時: <span className="font-mono">{new Date(data.created_at).toLocaleString()}</span>
	</div>
      )}
      </div>

      <div className="text-xs text-gray-400 font-mono mb-8 break-all">
        URI: {data.id}
      </div>

      <div className="mb-8">
        <h2 className="text-sm font-bold text-gray-500 uppercase tracking-wider mb-3">特徴・形質・関係性</h2>
        <div className="flex flex-wrap gap-2">
          {data.traits && data.traits.length > 0 ? (
            data.traits.map((t, i) => (
              <span key={i} className="inline-flex items-center gap-2 px-3 py-1.5 bg-gray-100 border border-gray-200 text-gray-800 rounded-md text-sm">
                <span className="font-bold text-blue-600">{t.predicate_label || "性質"}</span>
                <ArrowRight className="h-3 w-3 text-gray-400" />
                <span>{t.value_label}</span>
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

export default function OccurrenceDetail() {
  return (
    <main className="min-h-screen bg-gray-50 py-10 px-4">
      <Suspense fallback={<div className="text-center p-10">読み込み中...</div>}>
        <DetailContent />
      </Suspense>
    </main>
  );
}
