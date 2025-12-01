"use client";

import { useState } from "react";
import OntologySearch, { SearchResult } from "./OntologySearch";
import { Trash2, Send } from "lucide-react";

export default function OccurrenceForm() {
  // フォームの状態管理
  const [taxonLabel, setTaxonLabel] = useState("タヌキ"); // とりあえず初期値
  const [taxonID, setTaxonID] = useState("ncbi:34844");   // とりあえず初期値
  const [traits, setTraits] = useState<SearchResult[]>([]);
  const [remarks, setRemarks] = useState("");
  const [status, setStatus] = useState<"idle" | "submitting" | "success" | "error">("idle");

  // 形質が選ばれたときの処理
  const addTrait = (item: SearchResult) => {
    // 重複チェック
    if (!traits.find((t) => t.id === item.id)) {
      setTraits([...traits, item]);
    }
  };

  // 形質を削除する処理
  const removeTrait = (id: string) => {
    setTraits(traits.filter((t) => t.id !== id));
  };

  // 送信処理
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatus("submitting");

    // 送信するデータ (BackendのAPI仕様に合わせる)
    const payload = {
      taxon_id: taxonID,
      taxon_label: taxonLabel,
      traits: traits.map((t) => ({ id: t.id, label: t.label })),
      remarks: remarks,
    };

    try {
      const res = await fetch("http://localhost:8080/api/occurrences", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!res.ok) throw new Error("送信エラー");

      setStatus("success");
      setRemarks(""); // 成功したらメモだけクリア（続けて登録しやすいように）
      setTraits([]);
    } catch (error) {
      console.error(error);
      setStatus("error");
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6 bg-white p-6 rounded-xl shadow-lg border border-gray-100 max-w-2xl mx-auto">
      
      {/* 1. 生物名（今回は手入力） */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">生物名 (Taxon)</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={taxonLabel}
            onChange={(e) => setTaxonLabel(e.target.value)}
            className="flex-1 p-2 border border-gray-300 rounded text-black"
            placeholder="生物名"
            required
          />
          <input
            type="text"
            value={taxonID}
            onChange={(e) => setTaxonID(e.target.value)}
            className="w-32 p-2 border border-gray-300 rounded bg-gray-50 text-gray-600 text-sm font-mono"
            placeholder="ID"
            required
          />
        </div>
      </div>

      {/* 2. 形質（検索して追加） */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">特徴・形質 (Traits)</label>
        <OntologySearch onSelect={addTrait} />
        
        {/* 選ばれた形質のリスト表示 */}
        <div className="flex flex-wrap gap-2 mt-3">
          {traits.map((t) => (
            <span key={t.id} className="inline-flex items-center gap-1 px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm">
              {t.label}
              <button type="button" onClick={() => removeTrait(t.id)} className="text-blue-600 hover:text-red-500">
                <Trash2 className="h-4 w-4" />
              </button>
            </span>
          ))}
          {traits.length === 0 && <span className="text-gray-400 text-sm">（まだ選択されていません）</span>}
        </div>
      </div>

      {/* 3. 自由記述メモ */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">メモ (Remarks)</label>
        <textarea
          value={remarks}
          onChange={(e) => setRemarks(e.target.value)}
          className="w-full p-2 border border-gray-300 rounded h-24 text-black"
          placeholder="発見時の状況など..."
        />
      </div>

      {/* 送信ボタン */}
      <button
        type="submit"
        disabled={status === "submitting"}
        className="w-full py-3 bg-blue-600 hover:bg-blue-700 text-white font-bold rounded-lg flex justify-center items-center gap-2 transition-colors"
      >
        {status === "submitting" ? (
          "送信中..."
        ) : (
          <>
            <Send className="h-5 w-5" /> 登録する
          </>
        )}
      </button>

      {/* ステータス表示 */}
      {status === "success" && (
        <div className="p-3 bg-green-100 text-green-700 rounded text-center">
          ✅ 登録成功！データベースに保存されたのだ！
        </div>
      )}
      {status === "error" && (
        <div className="p-3 bg-red-100 text-red-700 rounded text-center">
          ❌ エラーが発生したのだ。Goサーバーは動いてる？
        </div>
      )}
    </form>
  );
}
