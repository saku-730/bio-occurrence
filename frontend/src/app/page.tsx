import OccurrenceForm from "@/components/OccurrenceForm";

export default function Home() {
  return (
    <main className="min-h-screen bg-gray-50 py-12 px-4">
      <div className="max-w-3xl mx-auto">
        <h1 className="text-3xl font-bold text-center text-gray-800 mb-8">
          生物多様性DB オカレンス登録
        </h1>
        <OccurrenceForm />
      </div>
    </main>
  );
}
