"use client";

import { useState, useEffect } from "react";
import OntologySearch, { SearchResult } from "./OntologySearch";
import { Trash2, Send, Save } from "lucide-react";
import { useRouter } from "next/navigation"; // 完了後の移動用

// 編集モード用のProps定義
type Props = {
  id?: string; // 更新対象のID (なければ新規登録)
  initialData?: {
    taxon_label: string;
    taxon_id: string; // 今は固定だけど将来用
    remarks: string;
    traits: SearchResult[];
  };
};

export default function OccurrenceForm({ id, initialData }: Props) {
  const router = useRouter();

  // 初期値があればそれを使う
  const [taxonLabel, setTaxonLabel] = useState(initialData?.taxon_label || "タヌキ");
  const [taxonID, setTaxonID] = useState(initialData?.taxon_id || "ncbi:34844");
  const [traits, setTraits] = useState<SearchResult[]>(initialData?.traits || []);
  const [remarks, setRemarks] = useState(initialData?.remarks || "");
  
  const [status, setStatus] = useState<"idle" | "submitting" | "success" | "error">("idle");

  const addTrait = (item: SearchResult) => {
    if (!traits.find((t) => t.id === item.id)) {
      setTraits([...traits, item]);
    }
  };

  const removeTrait = (id: string) => {
    setTraits(traits.filter((t) => t.id !== id));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatus("submitting");

    const payload = {
      taxon_id: taxonID,
      taxon_label: taxonLabel,
      traits: traits.map((t) => ({ id: t.id, label: t.label })),
      remarks: remarks,
    };

    try {
      // IDがあるなら PUT (更新)、なければ POST (新規)
      const url = id 
        ? `http://localhost:8080/api/occurrences/${id}`
        : "http://localhost:8080/api/occurrences";
      
      const method = id ? "PUT" : "POST";

      const res = await fetch(url, {
        method: method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!res.ok) throw new Error("送信エラー");

      setStatus("success");
      
      // 更新の場合は一覧に戻るなどの処理
      if (id) {
        setTimeout(() => router.push(`/occurrences/detail?id=${id}`), 1000);
      } else {
        // 新規の場合はフォームをクリア
        setRemarks("");
        setTraits([]);
      }

    } catch (error) {
      console.error(error);
      setStatus("error");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6 bg-white p-6 rounded-xl shadow-lg border border-gray-100 max-w-2xl mx-auto">
      {/* ... (入力フィールド部分は変更なしなので省略) ... */}
      
      {/* 1. 生物名 */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">生物名 (Taxon)</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={taxonLabel}
            onChange={(e) => setTaxonLabel(e.target.value)}
            className="flex-1 p-2 border border-gray-300 rounded text-black"
            required
          />
          <input
            type="text"
            value={taxonID}
            onChange={(e) => setTaxonID(e.target.value)}
            className="w-32 p-2 border border-gray-300 rounded bg-gray-50 text-gray-600 text-sm font-mono"
            required
          />
        </div>
      </div>

      {/* 2. 形質 */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">特徴・形質 (Traits)</label>
        <OntologySearch onSelect={addTrait} />
        <div className="flex flex-wrap gap-2 mt-3">
          {traits.map((t) => (
            <span key={t.id} className="inline-flex items-center gap-1 px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm">
              {t.label}
              <button type="button" onClick={() => removeTrait(t.id)} className="text-blue-600 hover:text-red-500">
                <Trash2 className="h-4 w-4" />
              </button>
            </span>
          ))}
        </div>
      </div>

      {/* 3. メモ */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">メモ (Remarks)</label>
        <textarea
          value={remarks}
          onChange={(e) => setRemarks(e.target.value)}
          className="w-full p-2 border border-gray-300 rounded h-24 text-black"
        />
      </div>

      {/* ボタンの文言を変える */}
      <button
        type="submit"
        disabled={status === "submitting"}
        className={`w-full py-3 font-bold rounded-lg flex justify-center items-center gap-2 transition-colors text-white ${
            id ? "bg-green-600 hover:bg-green-700" : "bg-blue-600 hover:bg-blue-700"
        }`}
      >
        {status === "submitting" ? "送信中..." : (
          id ? <><Save className="h-5 w-5" /> 更新を保存する</> : <><Send className="h-5 w-5" /> 登録する</>
        )}
      </button>

      {/* ... (ステータス表示も同じ) ... */}
      {status === "success" && (
        <div className="p-3 bg-green-100 text-green-700 rounded text-center">
          ✅ {id ? "更新成功！詳細ページに戻るのだ..." : "登録成功！データベースに保存されたのだ！"}
        </div>
      )}
    </form>
  );
}
