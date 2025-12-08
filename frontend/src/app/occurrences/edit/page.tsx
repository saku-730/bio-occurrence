"use client";

import { useEffect, useState, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import OccurrenceForm from "@/components/OccurrenceForm"; // 改造したフォームを使う

const API_URL = process.env.NEXT_PUBLIC_API_URL;

function EditContent() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id");
  const [initialData, setInitialData] = useState<any>(null);

  useEffect(() => {
    if (!id) return;
    // 既存データを取得してフォームの初期値にする
    fetch(`${API_URL}/api/occurrences/${id}`)
      .then((res) => res.json())
      .then((data) => {
        // APIのレスポンスをフォームの形に合わせる（traitsにuri等は不要なので）
        setInitialData({
            taxon_label: data.taxon_label,
            taxon_id: "ncbi:34844", // APIがまだ返してないので固定（後でAPI修正推奨）
            remarks: data.remarks,
            traits: data.traits,
        });
      });
  }, [id]);

  if (!initialData) return <div className="p-10 flex justify-center"><Loader2 className="animate-spin" /></div>;

  return (
    <div className="max-w-3xl mx-auto py-10 px-4">
      <h1 className="text-2xl font-bold mb-6 text-gray-800">データの編集</h1>
      {/* フォームにIDと初期データを渡す */}
      <OccurrenceForm id={id!} initialData={initialData} />
    </div>
  );
}

export default function EditPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <EditContent />
    </Suspense>
  );
}
