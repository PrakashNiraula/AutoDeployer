/** Long-sleep child used by bat→java kill tests. */
public class SleepJava {
    public static void main(String[] args) throws Exception {
        long ms = 600_000; // 10 minutes
        if (args.length > 0) {
            ms = Long.parseLong(args[0]);
        }
        System.out.println("SleepJava pid-start sleep_ms=" + ms);
        System.out.flush();
        Thread.sleep(ms);
        System.out.println("SleepJava done");
    }
}
