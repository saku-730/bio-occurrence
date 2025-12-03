"use client";

import { useState } from "react";
import OntologySearch, { SearchResult, TraitItem } from "./OntologySearch"; // ★修正: 型をまとめてインポート
import { Trash2, Send, Save, ArrowRight } from "lucide-react"; // ★修正: ArrowRightを追加
import { useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import TaxonSearch, { TaxonResult } from "./TaxonSearch";

// Props定義
type Props = {
  id?: string;
  initialData?: {
    taxon_label: string;
    taxon_id: string;
    remarks: string;
    traits: TraitItem[]; // ★修正: SearchResult[] ではなく TraitItem[] に変更
  };
};

export default function OccurrenceForm({ id, initialData }: Props) {
  const router = useRouter();
  const { token } = useAuth();

  // 初期値
  const [taxonLabel, setTaxonLabel] = useState(initialData?.taxon_label || "タヌキ");
  const [taxonID, setTaxonID] = useState(initialData?.taxon_id || "ncbi:34844");
  
  // ★修正: TraitItem[] として初期化
  const [traits, setTraits] = useState<TraitItem[]>(initialData?.traits || []);
  
  const [remarks, setRemarks] = useState(initialData?.remarks || "");
  const [status, setStatus] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [isPublic, setIsPublic] = useState(true);

  // 追加処理
  const addTrait = (item: TraitItem) => {
    // 重複チェック (述語と値のペアで判定)
    const exists = traits.some(
      (t) => t.predicateLabel === item.predicateLabel && t.valueLabel === item.valueLabel
    );
    if (!exists) {
      setTraits([...traits, item]);
    }
  };

  const removeTrait = (index: number) => {
    setTraits(traits.filter((_, i) => i !== index));
  };

  const handleTaxonSelect = (item: TaxonResult | null) => {
    if (item) {
      setTaxonLabel(item.label);
      setTaxonID(item.id.replace("_", ":")); // "NCBITaxon:34844" の形にしておくのが無難
    } else {
      setTaxonID(""); 
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!token) {
      alert("データを登録するにはログインが必要なのだ！");
      router.push("/login");
      return;
    }

    setStatus("submitting");

    // バックエンド (Go) の OccurrenceRequest 構造体に合わせる
    const payload = {
      taxon_id: taxonID,
      taxon_label: taxonLabel,
      traits: traits.map((t) => ({
        predicate_id: t.predicateID,       // Go: PredicateID
        predicate_label: t.predicateLabel, // Go: PredicateLabel
        value_id: t.valueID,               // Go: ValueID
        value_label: t.valueLabel          // Go: ValueLabel
      })),
      remarks: remarks,
      is_public: isPublic,
    };

    try {
      const url = id 
        ? `http://localhost:8080/api/occurrences/${id}`
        : "http://localhost:8080/api/occurrences";
      
      const method = id ? "PUT" : "POST";

      const res = await fetch(url, {
        method: method,
        headers: { 
            "Content-Type": "application/json",
            "Authorization": `Bearer ${token}`
        },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        console.error("Server Error:", res.status, await res.text());
        throw new Error("送信エラー");
      }

      setStatus("success");
      
      if (id) {
        setTimeout(() => router.push(`/occurrences/detail?id=${id}`), 1000);
      } else {
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
      
      {/* 1. 生物名 */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">
          生物名 (Taxon) <span className="text-xs font-normal text-gray-500">※同定限界</span>
        </label>

	<TaxonSearch 
	initialValue={taxonLabel} 
	onSelect={handleTaxonSelect} 
	/>

	{/* ID確認用（デバッグ用に見せてもいいし、hiddenにしてもいい） */}
	<div className="mt-1 text-xs text-gray-400 font-mono">
	ID: <input 
	  type="text" 
	  value={taxonID} 
	  readOnly
	  onChange={(e) => setTaxonID(e.target.value)} 
	  className="bg-transparent border-b border-gray-300 focus:outline-none w-40"
	  placeholder="自動入力されます"
	/>
	</div>
      </div>

      {/* 2. 特徴・形質 (トリプル入力) */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-2">特徴・形質・関係性</label>
        
        {/* リスト表示 */}
        <div className="flex flex-wrap gap-2 mb-3">
          {traits.map((t, index) => (
            <span key={index} className="inline-flex items-center gap-2 px-3 py-1.5 bg-gray-100 border border-gray-200 text-gray-800 rounded-md text-sm">
              <span className="font-bold text-blue-600">{t.predicateLabel}</span>
              <ArrowRight className="h-3 w-3 text-gray-400" />
              <span>{t.valueLabel}</span>
              <button type="button" onClick={() => removeTrait(index)} className="ml-1 text-gray-400 hover:text-red-500">
                <Trash2 className="h-4 w-4" />
              </button>
            </span>
          ))}
          {traits.length === 0 && <div className="text-sm text-gray-400 p-2">まだ追加されていません</div>}
        </div>

        {/* 検索コンポーネント */}
        <OntologySearch onAdd={addTrait} />
      </div>

      {/* 3. メモ */}
      <div>
        <label className="block text-sm font-bold text-gray-700 mb-1">メモ (Remarks)</label>
        <textarea
          value={remarks || ""}
          onChange={(e) => setRemarks(e.target.value)}
          className="w-full p-2 border border-gray-300 rounded h-24 text-black"
        />
      </div>

      {/* 4. 公開設定 */}
      <div className="flex items-center gap-2 bg-gray-50 p-3 rounded border border-gray-200">
        <input
          type="checkbox"
          id="isPublic"
          checked={isPublic}
          onChange={(e) => setIsPublic(e.target.checked)}
          className="w-5 h-5 text-blue-600 rounded focus:ring-blue-500"
        />
        <label htmlFor="isPublic" className="text-gray-700 font-medium cursor-pointer select-none">
          このデータを公開する（共有）
        </label>
      </div>
      <p className="text-xs text-gray-500 mb-4 pl-1">
        ※ チェックを外すと、あなた以外には表示されなくなるのだ（プライベートモード）。
      </p>

      {/* 送信ボタン */}
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

      {status === "success" && (
        <div className="p-3 bg-green-100 text-green-700 rounded text-center">
          ✅ {id ? "更新成功！詳細ページに戻るのだ..." : "登録成功！データベースに保存されたのだ！"}
        </div>
      )}
      {status === "error" && (
         <div className="p-3 bg-red-100 text-red-700 rounded text-center">
           ❌ 送信エラーなのだ。ログイン切れかも？
         </div>
       )}
    </form>
  );
}
