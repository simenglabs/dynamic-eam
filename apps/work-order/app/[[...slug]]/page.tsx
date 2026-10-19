import { ZonePage } from "@eam/ui";

export default function Page(props: { params: Promise<{ slug?: string[] }> }) {
  return <ZonePage zone="work-order" params={props.params} />;
}
